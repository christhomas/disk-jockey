package plugins

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SFTPPlugin implements the Plugin interface for SFTP-backed mounts
// Config expects: host, port, username, password, root

type SFTPPlugin struct{}

func (SFTPPlugin) Name() string { return "sftp" }

func (SFTPPlugin) Description() string {
	return "SFTP-backed remote filesystem mount"
}

func (SFTPPlugin) ConfigTemplate() PluginConfigTemplate {
	return PluginConfigTemplate{
		"host": PluginConfigField{
			Type:        "string",
			Description: "Remote SFTP server hostname",
			Required:    true,
		},
		"port": PluginConfigField{
			Type:        "string",
			Description: "SFTP port (default 22)",
			Required:    true,
		},
		"username": PluginConfigField{
			Type:        "string",
			Description: "Username for SFTP",
			Required:    true,
		},
		"password": PluginConfigField{
			Type:        "string",
			Description: "Password for SFTP (not secure, demo only)",
			Required:    false,
		},
		"use_ssh_agent": PluginConfigField{
			Type:        "bool",
			Description: "Use SSH agent for authentication (if available)",
			Required:    false,
		},
		"path": PluginConfigField{
			Type:        "string",
			Description: "Remote path prefix for all requests",
			Required:    true,
		},
	}
}

func (SFTPPlugin) New(mountName string, configSvc ConfigServiceIface) (Backend, error) {
	b := &SFTPBackend{mountName: mountName, configSvc: configSvc}
	if err := b.connect(); err != nil {
		return nil, err
	}
	return b, nil
}

// SFTPBackend implements Backend for a remote SFTP filesystem

type SFTPBackend struct {
	client    *sftp.Client
	mountName string
	configSvc ConfigServiceIface
	path      string // cached after connect
}

func (b *SFTPBackend) connect() error {
	cfg, ok := b.configSvc.GetMountConfig(b.mountName)
	if !ok {
		return fmt.Errorf("SFTPBackend: config for mount '%s' not found", b.mountName)
	}
	host, _ := cfg["host"].(string)
	port, _ := cfg["port"].(string)
	username, _ := cfg["username"].(string)
	password, _ := cfg["password"].(string)
	b.path = cfg["path"].(string)
	useAgent, _ := cfg["use_ssh_agent"].(bool)
	if host == "" || username == "" {
		return fmt.Errorf("missing required sftp config fields")
	}
	addr := host + ":" + port
	auths := []ssh.AuthMethod{}
	if password != "" {
		auths = append(auths, ssh.Password(password))
	}
	if useAgent {
		sshAgentSock := os.Getenv("SSH_AUTH_SOCK")
		if sshAgentSock != "" {
			agentConn, err := net.Dial("unix", sshAgentSock)
			if err == nil {
				auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(agentConn).Signers))
			}
		}
	}
	if len(auths) == 0 {
		return fmt.Errorf("no authentication method provided (set password or use_ssh_agent)")
	}
	sshConfig := &ssh.ClientConfig{
		User:            username,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // WARNING: for demo only
		Timeout:         5 * time.Second,
	}
	sshConn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("ssh dial failed: %w", err)
	}
	sftpClient, err := sftp.NewClient(sshConn)
	if err != nil {
		return fmt.Errorf("sftp client failed: %w", err)
	}
	b.client = sftpClient

	return nil
}

func (b *SFTPBackend) List(path string) ([]FileInfo, error) {
	absPath := b.path + path
	fmt.Println("SFTP List absPath:", absPath) // <-- Add this line
	files, err := b.client.ReadDir(absPath)
	if err != nil {
		return nil, err
	}
	var out []FileInfo
	for _, f := range files {
		out = append(out, FileInfo{
			Name:  f.Name(),
			IsDir: f.IsDir(),
			Size:  f.Size(),
		})
	}
	return out, nil
}

func (b *SFTPBackend) Read(path string) ([]byte, error) {
	absPath := b.path + path
	f, err := b.client.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (b *SFTPBackend) Write(path string, data []byte) error {
	absPath := b.path + path
	f, err := b.client.Create(absPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func (b *SFTPBackend) Delete(path string) error {
	absPath := b.path + path
	return b.client.Remove(absPath)
}

func (b *SFTPBackend) Close() error {
	return b.client.Close()
}

func (b *SFTPBackend) Reconnect() error {
	if b.client != nil {
		b.client.Close()
	}
	auths := []ssh.AuthMethod{}
	if len(auths) == 0 {
		return fmt.Errorf("no authentication method provided (set password or use_ssh_agent)")
	}
	sshConfig := &ssh.ClientConfig{
		User:            "", // USERNAME should be set from config in connect()
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // WARNING: for demo only
		Timeout:         5 * time.Second,
	}
	addr := "" // ADDR should be set from config in connect()
	sshConn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("ssh dial failed: %w", err)
	}
	sftpClient, err := sftp.NewClient(sshConn)
	if err != nil {
		return fmt.Errorf("sftp client failed: %w", err)
	}
	b.client = sftpClient
	return nil
}
