import Cocoa
import SwiftUI

class MenuBarController: NSObject {
    private var statusItem: NSStatusItem?
    private var settingsWindow: NSWindow?
    private var settingsWindowController: NSWindowController?
    
    override init() {
        super.init()
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
        if let button = statusItem?.button {
            button.image = NSImage(systemSymbolName: "externaldrive", accessibilityDescription: "Disk Jockey")
        }
        let menu = NSMenu()
        let aboutItem = NSMenuItem(title: "About", action: #selector(showAbout), keyEquivalent: "")
        aboutItem.target = self
        menu.addItem(aboutItem)
        let settingsItem = NSMenuItem(title: "Settings", action: #selector(showSettings), keyEquivalent: ",")
        settingsItem.target = self
        menu.addItem(settingsItem)
        let quitPanelItem = NSMenuItem(title: "Quit", action: #selector(showQuitPanel), keyEquivalent: "q")
        quitPanelItem.target = self
        menu.addItem(quitPanelItem)
        menu.addItem(NSMenuItem.separator())
        menu.delegate = self
        statusItem?.menu = menu
    }
    
    @objc func showAbout() {
        NotificationCenter.default.post(name: Notification.Name("ShowAboutPage"), object: nil)
    }
    
    @objc func showSettings() {
        NotificationCenter.default.post(name: Notification.Name("ShowSettingsPage"), object: nil)
    }
    
    @objc func showQuitPanel() {
        NotificationCenter.default.post(name: Notification.Name("ShowQuitPage"), object: nil)
    }
    
    @objc private func windowClosed(_ notification: Notification) {
        if let window = notification.object as? NSWindow, window == settingsWindow {
            settingsWindow = nil
        }
    }
}

extension MenuBarController: NSMenuDelegate {}
