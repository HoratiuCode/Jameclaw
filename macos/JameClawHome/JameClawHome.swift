import AppKit
import AVFoundation
import Foundation
import SwiftUI
import UniformTypeIdentifiers
import UserNotifications
import WebKit

extension Notification.Name {
    static let jameclawNewChat = Notification.Name("jameclaw.new-chat")
    static let jameclawResumeSession = Notification.Name("jameclaw.resume-session")
    static let jameclawHomeNavigation = Notification.Name("com.jameclaw.home.navigate")
    static let jameclawCommandPalette = Notification.Name("jameclaw.command-palette")
    static let jameclawTeamGrid = Notification.Name("jameclaw.team-grid")
    static let jameclawWorkspaceChanged = Notification.Name("jameclaw.workspace-changed")
    static let jameclawAutomationChanged = Notification.Name("jameclaw.automation-changed")
    static let jameclawFixedChatsChanged = Notification.Name("jameclaw.fixed-chats-changed")
}

private func authenticatedConsoleURL(
    port: Int,
    path: String = "/",
    queryItems: [URLQueryItem] = []
) -> URL {
    var components = URLComponents()
    components.scheme = "http"
    // The desktop launcher binds to IPv4 loopback. URLSession may otherwise
    // resolve localhost to ::1 first, where no launcher is listening.
    components.host = "127.0.0.1"
    components.port = port
    components.path = path
    let tokenURL = FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent(".jameclaw/launcher_access_token")
    var items = queryItems
    if let token = try? String(contentsOf: tokenURL, encoding: .utf8).trimmingCharacters(in: .whitespacesAndNewlines), !token.isEmpty {
        items.append(URLQueryItem(name: "access_token", value: token))
    }
    if !items.isEmpty {
        components.queryItems = items
    }
    return components.url ?? URL(string: "http://127.0.0.1:\(port)")!
}

// Native URLSession requests keep the launcher token in a cookie as well as
// in the bootstrap URL. This avoids depending on redirect/cookie persistence
// during the first connection attempt after the launcher starts.
private func authenticatedConsoleRequest(
    port: Int,
    path: String,
    method: String = "GET",
    queryItems: [URLQueryItem] = []
) -> URLRequest {
    authenticatedConsoleRequest(url: authenticatedConsoleURL(port: port, path: path, queryItems: queryItems), method: method)
}

private func authenticatedConsoleRequest(url: URL, method: String = "GET") -> URLRequest {
    var request = URLRequest(url: url)
    request.httpMethod = method
    let tokenURL = FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent(".jameclaw/launcher_access_token")
    if let token = try? String(contentsOf: tokenURL, encoding: .utf8).trimmingCharacters(in: .whitespacesAndNewlines), !token.isEmpty {
        request.setValue("jameclaw_launcher_session=\(token)", forHTTPHeaderField: "Cookie")
    }
    return request
}

private func authenticatedSessionURL(port: Int, id: String) -> URL {
    var components = URLComponents(url: authenticatedConsoleURL(port: port), resolvingAgainstBaseURL: false)!
    // Session keys from group channels can contain slashes and colons. They
    // are one API path parameter, not nested paths, so encode every reserved
    // character before asking the native launcher to retrieve the session.
    let allowed = CharacterSet.alphanumerics.union(CharacterSet(charactersIn: "-._~"))
    let encodedID = id.addingPercentEncoding(withAllowedCharacters: allowed) ?? id
    components.percentEncodedPath = "/api/sessions/\(encodedID)"
    return components.url ?? authenticatedConsoleURL(port: port, path: "/api/sessions")
}

// Keep every native page on the same design tokens. The selected full-design
// preset writes its accent to UserDefaults; resolving it here means supporting
// pages that use the shared brand token update together with Chat and Settings.
private enum JameBrand {
    static var orange: Color { selectedAccent(soft: false) }
    static var orangeSoft: Color { selectedAccent(soft: true) }
    static let ink = Color(red: 0.035, green: 0.035, blue: 0.038)
    static let panel = Color(red: 0.072, green: 0.072, blue: 0.078)
    static let elevated = Color(red: 0.105, green: 0.105, blue: 0.115)
    static let paper = Color(red: 0.97, green: 0.965, blue: 0.95)
    static let muted = Color.white.opacity(0.58)
    static let rule = Color.white.opacity(0.10)

    private static func selectedAccent(soft: Bool) -> Color {
        switch UserDefaults.standard.string(forKey: "launcher.design.accent") ?? "theme" {
        case "coral":
            return soft ? Color(red: 1.00, green: 0.49, blue: 0.45) : Color(red: 0.95, green: 0.38, blue: 0.34)
        case "blue":
            return soft ? Color(red: 0.40, green: 0.68, blue: 1.00) : Color(red: 0.28, green: 0.58, blue: 0.98)
        case "mint":
            return soft ? Color(red: 0.38, green: 0.88, blue: 0.71) : Color(red: 0.23, green: 0.78, blue: 0.62)
        case "violet":
            return soft ? Color(red: 0.74, green: 0.57, blue: 1.00) : Color(red: 0.64, green: 0.46, blue: 0.96)
        case "gold":
            return soft ? Color(red: 1.00, green: 0.77, blue: 0.34) : Color(red: 0.94, green: 0.68, blue: 0.22)
        default:
            return soft ? Color(red: 1.00, green: 0.45, blue: 0.12) : Color(red: 1.00, green: 0.37, blue: 0.04)
        }
    }
}

private struct SquareSegmentedPicker<Value: Hashable>: View {
    let options: [(label: String, value: Value)]
    @Binding var selection: Value
    @Environment(\.colorScheme) private var colorScheme

    private var panel: Color { colorScheme == .light ? .white : JameBrand.elevated }
    private var primary: Color { colorScheme == .light ? JameBrand.ink : JameBrand.paper }
    private var rule: Color { colorScheme == .light ? JameBrand.ink.opacity(0.14) : JameBrand.rule }

    var body: some View {
        HStack(spacing: 0) {
            ForEach(options.indices, id: \.self) { index in
                let option = options[index]
                Button {
                    selection = option.value
                } label: {
                    Text(option.label)
                        .font(.caption.weight(.semibold))
                        .frame(maxWidth: .infinity)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 7)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .foregroundStyle(selection == option.value ? JameBrand.ink : primary)
                .background(selection == option.value ? JameBrand.orange : panel)
                .overlay(Rectangle().stroke(rule, lineWidth: 1))
            }
        }
        .overlay(Rectangle().stroke(JameBrand.orange.opacity(0.42), lineWidth: 1))
    }
}

/// A deliberately branded trigger for the places where a compact choice is
/// useful. The menu itself remains the accessible native macOS menu, while
/// the control that lives in the JameClaw chrome no longer reads as a default
/// SwiftUI picker.
private struct JameDropdownPicker<Value: Hashable>: View {
    let label: String
    let options: [(label: String, value: Value)]
    @Binding var selection: Value
    var minWidth: CGFloat = 150

    private var selectedLabel: String {
        options.first(where: { $0.value == selection })?.label ?? label
    }

    var body: some View {
        Menu {
            ForEach(options.indices, id: \.self) { index in
                let option = options[index]
                Button {
                    selection = option.value
                } label: {
                    Label(
                        option.label,
                        systemImage: selection == option.value ? "checkmark" : "circle"
                    )
                }
            }
        } label: {
            HStack(spacing: 8) {
                Rectangle()
                    .fill(JameBrand.orange)
                    .frame(width: 3, height: 16)
                Text(selectedLabel)
                    .font(.system(size: 11, weight: .semibold, design: .rounded))
                    .lineLimit(1)
                Spacer(minLength: 8)
                Image(systemName: "chevron.up.chevron.down")
                    .font(.system(size: 9, weight: .bold))
                    .foregroundStyle(JameBrand.orange)
            }
            .foregroundStyle(JameBrand.paper)
            .padding(.leading, 8)
            .padding(.trailing, 9)
            .frame(minWidth: minWidth, minHeight: 30)
            .background(JameBrand.elevated, in: Rectangle())
            .overlay(Rectangle().stroke(JameBrand.rule, lineWidth: 1))
            .contentShape(Rectangle())
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
        .fixedSize(horizontal: false, vertical: true)
        .accessibilityLabel(label)
        .accessibilityValue(selectedLabel)
    }
}

private struct SquareTextFieldStyle: TextFieldStyle {
    @Environment(\.colorScheme) private var colorScheme

    func _body(configuration: TextField<Self._Label>) -> some View {
        configuration
            .padding(.horizontal, 9)
            .padding(.vertical, 6)
            .background(colorScheme == .light ? Color.white : JameBrand.elevated)
            .overlay(Rectangle().stroke(colorScheme == .light ? JameBrand.ink.opacity(0.14) : JameBrand.rule, lineWidth: 1))
    }
}

private struct WindowOpacityConfigurator: NSViewRepresentable {
    let opacity: Double

    func makeNSView(context: Context) -> NSView {
        let view = NSView(frame: .zero)
        apply(to: view)
        return view
    }

    func updateNSView(_ nsView: NSView, context: Context) {
        apply(to: nsView)
    }

    private func apply(to view: NSView) {
        DispatchQueue.main.async {
            view.window?.alphaValue = CGFloat(min(max(opacity, 0.65), 1.0))
        }
    }
}

@MainActor
private final class JameMainWindowRegistry {
    static let shared = JameMainWindowRegistry()
    var window: NSWindow?
    var fallbackWindowController: NSWindowController?
    var preferredFrame: NSRect?
    weak var observedWindow: NSWindow?
    var liveResizeObserver: NSObjectProtocol?
}

private let jameMainWindowTitle = "JameClaw Desktop"

@MainActor
private func selectDesktopSection(
    _ section: DesktopSection?,
    selection: Binding<DesktopSection?>
) {
    let preservedFrame = JameMainWindowRegistry.shared.window?.frame
    JameMainWindowRegistry.shared.preferredFrame = preservedFrame
    selection.wrappedValue = section

    guard let preservedFrame else { return }
    // Some detail views (especially Chat's composer and canvas) report a
    // larger fitting size than list-based pages. Navigation must never resize
    // a window the user already sized, so restore its frame after SwiftUI's
    // immediate and deferred layout passes.
    for delay in [0.0, 0.08, 0.25, 0.6, 1.2, 2.2] {
        DispatchQueue.main.asyncAfter(deadline: .now() + delay) {
            guard let window = JameMainWindowRegistry.shared.window else { return }
            window.minSize = NSSize(width: 660, height: 440)
            window.setFrame(preservedFrame, display: true, animate: false)
        }
    }
}

private struct WindowBehaviorConfigurator: NSViewRepresentable {
    func makeNSView(context: Context) -> NSView {
        let view = NSView(frame: .zero)
        apply(to: view)
        return view
    }

    func updateNSView(_ nsView: NSView, context: Context) {
        apply(to: nsView)
    }

    private func apply(to view: NSView) {
        DispatchQueue.main.async {
            guard let window = view.window else { return }
            window.styleMask.formUnion([.titled, .closable, .miniaturizable, .resizable])
            window.isMovable = true
            window.isMovableByWindowBackground = true
            window.title = jameMainWindowTitle
            // SwiftUI may remove a closed WindowGroup window from
            // `NSApp.windows`. Retain the actual main window so Finder, Dock,
            // menu-bar, and notification reopen requests can show it again.
            window.isReleasedWhenClosed = false
            if let fallbackWindow = JameMainWindowRegistry.shared.fallbackWindowController?.window,
               fallbackWindow !== window {
                fallbackWindow.orderOut(nil)
                fallbackWindow.close()
                JameMainWindowRegistry.shared.fallbackWindowController = nil
            }
            JameMainWindowRegistry.shared.window = window
            window.minSize = NSSize(width: 660, height: 440)
            window.setFrameAutosaveName("JameClawMainWindow")

            let registry = JameMainWindowRegistry.shared
            if registry.preferredFrame == nil {
                let visibleFrame = (window.screen ?? NSScreen.main)?.visibleFrame
                    ?? NSRect(x: 0, y: 0, width: 1120, height: 720)
                let savedFrame = UserDefaults.standard.string(forKey: "launcher.mainWindowFrame")
                    .map(NSRectFromString)
                let savedFrameIsUsable = savedFrame.map {
                    $0.width >= 660 && $0.height >= 440 && $0.intersects(visibleFrame)
                } ?? false
                var launchFrame = savedFrameIsUsable
                    ? savedFrame!
                    : NSRect(
                        x: visibleFrame.midX - 560,
                        y: visibleFrame.midY - 360,
                        width: min(1120, visibleFrame.width),
                        height: min(720, visibleFrame.height)
                    )
                launchFrame.size.width = min(launchFrame.width, visibleFrame.width)
                launchFrame.size.height = min(launchFrame.height, visibleFrame.height)
                launchFrame.origin.x = min(max(launchFrame.minX, visibleFrame.minX), visibleFrame.maxX - launchFrame.width)
                launchFrame.origin.y = min(max(launchFrame.minY, visibleFrame.minY), visibleFrame.maxY - launchFrame.height)
                registry.preferredFrame = launchFrame
                window.setFrame(launchFrame, display: true, animate: false)
                for delay in [0.25, 0.8, 1.5, 2.5] {
                    DispatchQueue.main.asyncAfter(deadline: .now() + delay) {
                        guard registry.window === window,
                              registry.preferredFrame == launchFrame else { return }
                        window.setFrame(launchFrame, display: true, animate: false)
                    }
                }
            }

            if registry.observedWindow !== window {
                if let observer = registry.liveResizeObserver {
                    NotificationCenter.default.removeObserver(observer)
                }
                registry.observedWindow = window
                registry.liveResizeObserver = NotificationCenter.default.addObserver(
                    forName: NSWindow.didEndLiveResizeNotification,
                    object: window,
                    queue: .main
                ) { [weak window] _ in
                    guard let window else { return }
                    Task { @MainActor in
                        JameMainWindowRegistry.shared.preferredFrame = window.frame
                        UserDefaults.standard.set(
                            NSStringFromRect(window.frame),
                            forKey: "launcher.mainWindowFrame"
                        )
                    }
                }
            }
        }
    }
}

@MainActor
private func bootstrapJameMainWindow() {
    if let existingWindow = NSApp.windows.first(where: {
        $0.title == jameMainWindowTitle && ($0.isVisible || $0.isMiniaturized)
    }) {
        JameMainWindowRegistry.shared.window = existingWindow
        return
    }

    if let fallbackWindow = JameMainWindowRegistry.shared.fallbackWindowController?.window {
        JameMainWindowRegistry.shared.window = fallbackWindow
        fallbackWindow.makeKeyAndOrderFront(nil)
        return
    }

    let controller = NSHostingController(rootView: JameStandaloneRootView())
    let window = NSWindow(
        contentRect: NSRect(x: 0, y: 0, width: 1120, height: 720),
        styleMask: [.titled, .closable, .miniaturizable, .resizable],
        backing: .buffered,
        defer: false
    )
    window.title = jameMainWindowTitle
    window.contentViewController = controller
    window.isReleasedWhenClosed = false
    window.minSize = NSSize(width: 660, height: 440)
    window.setFrameAutosaveName("JameClawMainWindow")
    window.center()

    let windowController = NSWindowController(window: window)
    JameMainWindowRegistry.shared.window = window
    JameMainWindowRegistry.shared.fallbackWindowController = windowController
    windowController.showWindow(nil)
    window.makeKeyAndOrderFront(nil)
    NSApp.activate(ignoringOtherApps: true)
    NSLog("[JameWindow] created fallback; visible=%@", window.isVisible.description)
}

final class HomeAppDelegate: NSObject, NSApplicationDelegate, UNUserNotificationCenterDelegate {
    private var visibilityTimer: Timer?

    func applicationDidFinishLaunching(_ notification: Notification) {
        // JameClaw Desktop is the visible application. Its Go server and
        // gateway run as helper executables inside this same app bundle.
        let activationChanged = NSApp.setActivationPolicy(.regular)
        NSLog("[JameWindow] did finish launching; regular activation=%@", activationChanged.description)
        NativeLaunchCoordinator.shared.startIfNeeded()

        let center = UNUserNotificationCenter.current()
        center.delegate = self
        let key = "jame.notifications.taskCompletion"
        let notificationsEnabled = UserDefaults.standard.object(forKey: key) == nil || UserDefaults.standard.bool(forKey: key)
        if notificationsEnabled {
            center.requestAuthorization(options: [.alert, .sound]) { _, _ in }
        }

        // Do not wait for SwiftUI restoration before making Jame usable. The
        // hosting fallback is replaced automatically as soon as WindowGroup
        // installs its normal window.
        DispatchQueue.main.async {
            self.ensureMainWindowExists()
            self.revealMainWindow()
        }

        // SwiftUI mounts the WindowGroup just after the app delegate launch
        // callback. Reveal twice so a cold launch and a restored process both
        // end with the real main window in front.
        for delay in [0.2, 0.8, 2.0, 4.0, 6.0, 8.0, 10.0] {
            DispatchQueue.main.asyncAfter(deadline: .now() + delay) {
                self.ensureMainWindowExists()
                self.revealMainWindow()
            }
        }

        visibilityTimer = Timer.scheduledTimer(withTimeInterval: 2.0, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                guard let self else { return }
                let mainWindows = NSApp.windows.filter { $0.title == jameMainWindowTitle }
                let hasUsableWindow = mainWindows.contains { $0.isVisible || $0.isMiniaturized }
                guard !hasUsableWindow else { return }
                self.ensureMainWindowExists()
                self.revealMainWindow()
            }
        }
    }

    func applicationWillTerminate(_ notification: Notification) {
        visibilityTimer?.invalidate()
        visibilityTimer = nil
        NativeLaunchCoordinator.shared.stop()
    }

    func applicationShouldSaveApplicationState(_ sender: NSApplication) -> Bool {
        // This is a single-window app. Persisting a closed WindowGroup causes
        // the next launch to start as a windowless background process.
        false
    }

    func applicationShouldRestoreApplicationState(_ sender: NSApplication) -> Bool {
        false
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        // The launcher and gateway are separate processes. Terminating only
        // this native UI after its last window closes prevents macOS from
        // leaving a windowless Jame process that cannot be reopened cleanly.
        true
    }

    func applicationDidBecomeActive(_ notification: Notification) {
        revealMainWindow()
    }

    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        revealMainWindow()
        return true
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        completionHandler([.banner, .sound])
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        revealMainWindow()
        completionHandler()
    }

    private func revealMainWindow() {
        DispatchQueue.main.async {
            guard let window = NSApp.windows.first(where: {
                $0.title == jameMainWindowTitle && ($0.isVisible || $0.isMiniaturized)
            })
                ?? JameMainWindowRegistry.shared.window else { return }
            JameMainWindowRegistry.shared.window = window
            let visibleFrame = (window.screen ?? NSScreen.main)?.visibleFrame.insetBy(dx: 12, dy: 12)
            if let visibleFrame {
                var frame = window.frame
                frame.size.width = min(frame.width, visibleFrame.width)
                frame.size.height = min(frame.height, visibleFrame.height)
                frame.origin.x = min(max(frame.origin.x, visibleFrame.minX), visibleFrame.maxX - frame.width)
                frame.origin.y = min(max(frame.origin.y, visibleFrame.minY), visibleFrame.maxY - frame.height)
                window.setFrame(frame, display: true)
            } else {
                window.center()
            }
            window.deminiaturize(nil)
            window.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
        }
    }

    @MainActor
    private func ensureMainWindowExists() {
        NSLog("[JameWindow] ensure; windows=%ld", NSApp.windows.count)
        // SwiftUI can occasionally restore a WindowGroup as an active,
        // windowless process. Create the same root view in a normal AppKit
        // window so launching Jame always produces visible UI.
        bootstrapJameMainWindow()
    }
}

private struct LauncherSettings: Codable {
    var port: Int = 18800
    var `public`: Bool = false
    var allowedCIDRs: [String] = []

    enum CodingKeys: String, CodingKey {
        case port
        case `public`
        case allowedCIDRs = "allowed_cidrs"
    }
}

private enum LauncherTheme: String, CaseIterable, Identifiable {
    case light
    case dark
    case system

    var id: String { rawValue }
    var label: String {
        switch self {
        case .light: return "Light"
        case .dark: return "Dark"
        case .system: return "Match This Mac"
        }
    }
    var preferredColorScheme: ColorScheme? {
        switch self {
        case .light: return .light
        case .dark: return .dark
        case .system: return nil
        }
    }
    func resolved(for systemScheme: ColorScheme) -> LauncherTheme {
        self == .system ? (systemScheme == .dark ? .dark : .light) : self
    }
    var accent: Color {
        switch self {
        case .light: return Color(red: 0.72, green: 0.18, blue: 0.15)
        case .dark, .system: return JameBrand.orange
        }
    }
    var background: Color {
        switch self {
        case .light: return Color(red: 0.96, green: 0.97, blue: 0.98)
        case .dark, .system: return JameBrand.ink
        }
    }
    var panel: Color {
        switch self {
        case .light: return .white
        case .dark, .system: return JameBrand.panel
        }
    }
    var text: Color { self == .light ? Color(red: 0.13, green: 0.15, blue: 0.18) : JameBrand.paper }
}

private func launcherThemePreference(from rawValue: String) -> LauncherTheme {
    if let theme = LauncherTheme(rawValue: rawValue) { return theme }
    // Preserve pre-update dark choices such as Terminal, Midnight, Forest,
    // and Lavender when migrating to the simpler Light/Dark/Mac selector.
    return rawValue == "light" ? .light : .dark
}

private enum LauncherAccent: String, CaseIterable, Identifiable {
    case theme
    case coral
    case blue
    case mint
    case violet
    case gold

    var id: String { rawValue }
    var label: String { rawValue == "theme" ? "Match theme" : rawValue.capitalized }
    var color: Color? {
        switch self {
        case .theme: return nil
        case .coral: return Color(red: 0.95, green: 0.38, blue: 0.34)
        case .blue: return Color(red: 0.28, green: 0.58, blue: 0.98)
        case .mint: return Color(red: 0.23, green: 0.78, blue: 0.62)
        case .violet: return Color(red: 0.64, green: 0.46, blue: 0.96)
        case .gold: return Color(red: 0.94, green: 0.68, blue: 0.22)
        }
    }
}

private func launcherAccentPreference(from rawValue: String, theme: LauncherTheme) -> Color {
    (LauncherAccent(rawValue: rawValue) ?? .theme).color ?? theme.accent
}

private enum DocumentApprovalPolicy: String, CaseIterable, Identifiable {
    case alwaysAsk
    case outsideWorkspace
    case explicitSelection
    case workspaceOnly
    case yolo

    var id: String { rawValue }
    var label: String {
        switch self {
        case .alwaysAsk: return "Ask every time"
        case .outsideWorkspace: return "Ask outside workspace"
        case .explicitSelection: return "Selecting an item approves it"
        case .workspaceOnly: return "Current workspace only"
        case .yolo: return "YOLO — Never ask"
        }
    }
    var detail: String {
        switch self {
        case .alwaysAsk:
            return "Show an approval before Jame changes to any folder or document."
        case .outsideWorkspace:
            return "Work inside the current workspace without interruption; ask before using anything outside it."
        case .explicitSelection:
            return "A Finder drop or a click in Terminal + Documents counts as approval for that location."
        case .workspaceOnly:
            return "Block document access outside the current workspace until you change this safety policy."
        case .yolo:
            return "Never show document approval prompts and allow file tools outside the active workspace. Use only when you fully trust the agent and task."
        }
    }
}

@MainActor
private func approveDocumentAccess(to targetURL: URL, action: String) -> Bool {
    let rawPolicy = UserDefaults.standard.string(forKey: "launcher.safety.documentApprovalPolicy")
        ?? DocumentApprovalPolicy.outsideWorkspace.rawValue
    let policy = DocumentApprovalPolicy(rawValue: rawPolicy) ?? .outsideWorkspace
    let target = targetURL.standardizedFileURL.resolvingSymlinksInPath()
    let workspace = jameWorkspaceURL().standardizedFileURL.resolvingSymlinksInPath()
    let isInsideWorkspace = target.path == workspace.path || target.path.hasPrefix(workspace.path + "/")

    if policy == .workspaceOnly && !isInsideWorkspace {
        let alert = NSAlert()
        alert.alertStyle = .warning
        alert.messageText = "Document access blocked"
        alert.informativeText = "JameClaw is restricted to \(workspace.path). Change the document approval policy in Settings > Safety to use this location."
        alert.addButton(withTitle: "OK")
        alert.runModal()
        return false
    }

    let shouldAsk = policy == .alwaysAsk || (policy == .outsideWorkspace && !isInsideWorkspace)
    guard shouldAsk else { return true }
    let alert = NSAlert()
    alert.alertStyle = .informational
    alert.messageText = "Allow JameClaw document access?"
    alert.informativeText = "Jame wants to \(action):\n\n\(target.path)\n\nApproving sets the containing folder as the restricted agent workspace."
    alert.addButton(withTitle: "Allow")
    alert.addButton(withTitle: "Cancel")
    return alert.runModal() == .alertFirstButtonReturn
}

private enum ChatDensity: String, CaseIterable, Identifiable {
    case compact
    case comfortable
    case spacious

    var id: String { rawValue }
    var label: String { rawValue.capitalized }
    var messageSpacing: CGFloat { self == .compact ? 7 : self == .comfortable ? 12 : 18 }
    var messagePadding: CGFloat { self == .compact ? 9 : self == .comfortable ? 12 : 16 }
    var contentPadding: CGFloat { self == .compact ? 12 : self == .comfortable ? 18 : 26 }
}

private enum MessageSurface: String, CaseIterable, Identifiable {
    case cards
    case minimal

    var id: String { rawValue }
    var label: String { self == .cards ? "Cards" : "Minimal" }
}

private struct NativeDesignConfiguration {
    let theme: LauncherTheme
    let accent: LauncherAccent
    let density: ChatDensity
    let surface: MessageSurface
    let fontScale: Double
    let windowOpacity: Double
    let teamGlow: Bool
}

private enum NativeDesignPreset: String, CaseIterable, Identifiable {
    case jameDark
    case jameLight
    case nordicFrost
    case sepiaReading
    case cyberpunkNeon
    case forestFocus
    case sunsetStudio
    case custom

    var id: String { rawValue }

    var label: String {
        switch self {
        case .jameDark: return "Jame Dark"
        case .jameLight: return "Jame Light"
        case .nordicFrost: return "Nordic Frost"
        case .sepiaReading: return "Sepia Reading"
        case .cyberpunkNeon: return "Cyberpunk Neon"
        case .forestFocus: return "Forest Focus"
        case .sunsetStudio: return "Sunset Studio"
        case .custom: return "Custom design"
        }
    }

    var detail: String {
        switch self {
        case .jameDark: return "Dark · Orange · Compact · Cards"
        case .jameLight: return "Light · Coral · Comfortable · Cards"
        case .nordicFrost: return "Dark · Blue · Comfortable · Minimal"
        case .sepiaReading: return "Light · Gold · Spacious · Large text"
        case .cyberpunkNeon: return "Dark · Violet · Compact · Minimal"
        case .forestFocus: return "Dark · Mint · Comfortable · Cards"
        case .sunsetStudio: return "Dark · Coral · Spacious · Large text"
        case .custom: return "Fine-tuned with the individual controls below"
        }
    }

    var configuration: NativeDesignConfiguration? {
        switch self {
        case .jameDark:
            return NativeDesignConfiguration(theme: .dark, accent: .theme, density: .compact, surface: .cards, fontScale: 1.0, windowOpacity: 1.0, teamGlow: true)
        case .jameLight:
            return NativeDesignConfiguration(theme: .light, accent: .coral, density: .comfortable, surface: .cards, fontScale: 1.0, windowOpacity: 1.0, teamGlow: true)
        case .nordicFrost:
            return NativeDesignConfiguration(theme: .dark, accent: .blue, density: .comfortable, surface: .minimal, fontScale: 1.0, windowOpacity: 0.95, teamGlow: false)
        case .sepiaReading:
            return NativeDesignConfiguration(theme: .light, accent: .gold, density: .spacious, surface: .cards, fontScale: 1.15, windowOpacity: 1.0, teamGlow: false)
        case .cyberpunkNeon:
            return NativeDesignConfiguration(theme: .dark, accent: .violet, density: .compact, surface: .minimal, fontScale: 0.88, windowOpacity: 0.9, teamGlow: true)
        case .forestFocus:
            return NativeDesignConfiguration(theme: .dark, accent: .mint, density: .comfortable, surface: .cards, fontScale: 1.0, windowOpacity: 0.95, teamGlow: false)
        case .sunsetStudio:
            return NativeDesignConfiguration(theme: .dark, accent: .coral, density: .spacious, surface: .cards, fontScale: 1.15, windowOpacity: 0.95, teamGlow: true)
        case .custom:
            return nil
        }
    }
}

@MainActor
final class LauncherSettingsStore: ObservableObject {
    @Published var port = "18800"
    @Published var lanAccess = false
    @Published var allowedCIDRs = ""
    @Published var saveStatus = ""

    private let configURL = FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent(".jameclaw/launcher-config.json")

    init() { load() }

    func load() {
        guard let data = try? Data(contentsOf: configURL),
              let settings = try? JSONDecoder().decode(LauncherSettings.self, from: data) else { return }
        port = String(settings.port)
        lanAccess = settings.public
        allowedCIDRs = settings.allowedCIDRs.joined(separator: "\n")
    }

    func save() {
        guard let portNumber = Int(port), (1...65535).contains(portNumber) else {
            saveStatus = "Enter a port between 1 and 65535."
            return
        }
        let cidrs = allowedCIDRs.split(whereSeparator: \.isNewline).map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }
        let settings = LauncherSettings(port: portNumber, public: lanAccess, allowedCIDRs: cidrs)
        do {
            try FileManager.default.createDirectory(at: configURL.deletingLastPathComponent(), withIntermediateDirectories: true)
            let encoder = JSONEncoder()
            encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
            try encoder.encode(settings).write(to: configURL, options: .atomic)
            saveStatus = "Saved. Restart Jame for network changes to take effect."
        } catch {
            saveStatus = "Could not save settings."
        }
    }
}

private struct JameStandaloneRootView: View {
    @State private var selectedSection: DesktopSection? = .chat

    var body: some View {
        JameRootView(selectedSection: $selectedSection)
    }
}

private func migrateLegacyNativeDefaultsIfNeeded() {
    guard let bundleID = Bundle.main.bundleIdentifier,
          bundleID == "com.jameclaw.launcher" else { return }
    let marker = "jameclaw.defaults.migrated-from-home"
    let defaults = UserDefaults.standard
    guard !defaults.bool(forKey: marker) else { return }

    let legacy = defaults.persistentDomain(forName: "com.jameclaw.home") ?? [:]
    var current = defaults.persistentDomain(forName: bundleID) ?? [:]
    for (key, value) in legacy where current[key] == nil {
        current[key] = value
    }
    current[marker] = true
    defaults.setPersistentDomain(current, forName: bundleID)
}

@main
struct JameClawHomeApp: App {
    @NSApplicationDelegateAdaptor(HomeAppDelegate.self) private var appDelegate
    @State private var selectedSection: DesktopSection? = .chat

    init() {
        migrateLegacyNativeDefaultsIfNeeded()
        // This initializer runs even when SwiftUI restores no WindowGroup
        // scene. Bootstrap the visible AppKit host independently of scene
        // restoration; WindowBehaviorConfigurator swaps in the normal window
        // if SwiftUI creates one afterward.
        DispatchQueue.main.async {
            NSApp.setActivationPolicy(.regular)
            bootstrapJameMainWindow()
        }
    }

    var body: some Scene {
        WindowGroup { JameRootView(selectedSection: $selectedSection) }
            // A content-sized window disables the standard macOS full-screen
            // control. Keep the desktop window resizable so it can enter
            // full screen from the title bar, the toolbar, or ⌃⌘F.
            // Let AppKit own interactive resizing. SwiftUI's contentMinSize
            // mode can animate through a negative intermediate size when the
            // NavigationSplitView updates, which AppKit treats as a fatal
            // invalid-geometry error.
            .windowResizability(.automatic)
            .defaultSize(width: 1120, height: 720)
            .commands {
                CommandGroup(after: .newItem) {
                    Button("Quick Actions…") {
                        NotificationCenter.default.post(name: .jameclawCommandPalette, object: nil)
                    }
                    .keyboardShortcut("k", modifiers: [.command])

                    Button("New Chat") {
                        selectDesktopSection(.chat, selection: $selectedSection)
                        NotificationCenter.default.post(name: .jameclawNewChat, object: nil)
                    }
                    .keyboardShortcut("n", modifiers: [.command])
                }

                CommandMenu("Automations") {
                    Button("Show Automations") {
                        selectDesktopSection(.automations, selection: $selectedSection)
                    }
                    .keyboardShortcut("a", modifiers: [.command, .option])
                }

                CommandMenu("View") {
                    Button("Team Grid…") {
                        selectDesktopSection(.agent, selection: $selectedSection)
                        DispatchQueue.main.asyncAfter(deadline: .now() + 0.25) {
                            NotificationCenter.default.post(name: .jameclawTeamGrid, object: nil)
                        }
                    }
                    .keyboardShortcut("g", modifiers: [.command, .option])

                    Divider()
                    ForEach(DesktopSection.allCases) { section in
                        Button(section.title) {
                            selectDesktopSection(section, selection: $selectedSection)
                        }
                        .keyboardShortcut(section.menuShortcut, modifiers: [.command, .option])
                    }
                }
            }
    }
}

@MainActor
private final class NativeDesktopNavigationStore: ObservableObject {
    @Published private(set) var hasAutomations = false
    @Published private(set) var hasFixedChats = false
    @Published private(set) var messagingPlatformLabel = "No messaging connected"

    private let port: Int
    private var isRefreshing = false

    init(port: Int) {
        self.port = port
    }

    func monitor() async {
        while !Task.isCancelled {
            await refresh()
            try? await Task.sleep(for: .seconds(5))
        }
    }

    func refresh() async {
        guard !isRefreshing else { return }
        isRefreshing = true
        defer { isRefreshing = false }

        await loadAutomationVisibility()
        await loadFixedChatsVisibility()
        await loadMessagingPlatforms()
    }

    private func loadFixedChatsVisibility() async {
        do {
            let (data, response) = try await URLSession.shared.data(
                for: authenticatedConsoleRequest(
                    port: port,
                    path: "/api/sessions",
                    queryItems: [
                        URLQueryItem(name: "offset", value: "0"),
                        URLQueryItem(name: "limit", value: "10000"),
                    ]
                )
            )
            guard let http = response as? HTTPURLResponse,
                  (200..<300).contains(http.statusCode),
                  let sessions = try JSONSerialization.jsonObject(with: data) as? [[String: Any]] else { return }
            hasFixedChats = sessions.contains { $0["pinned"] as? Bool == true }
        } catch {
            // Preserve the last known navigation state during gateway restarts.
        }
    }

    private func loadAutomationVisibility() async {
        do {
            let (data, response) = try await URLSession.shared.data(
                for: authenticatedConsoleRequest(port: port, path: "/api/automation")
            )
            guard let http = response as? HTTPURLResponse,
                  (200..<300).contains(http.statusCode),
                  let body = try JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let items = body["items"] as? [Any] else { return }
            hasAutomations = !items.isEmpty
        } catch {
            // Preserve the last known navigation state during gateway restarts.
        }
    }

    private func loadMessagingPlatforms() async {
        do {
            let (data, response) = try await URLSession.shared.data(
                for: authenticatedConsoleRequest(port: port, path: "/api/gateway/status")
            )
            guard let http = response as? HTTPURLResponse,
                  (200..<300).contains(http.statusCode),
                  let body = try JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let channels = body["channels"] as? [String: Any] else {
                messagingPlatformLabel = "No messaging connected"
                return
            }

            let activePlatforms = Self.messagingChannels.compactMap { channel -> String? in
                guard let status = channels[channel.key] as? [String: Any],
                      status["running"] as? Bool == true else { return nil }
                return channel.name
            }
            messagingPlatformLabel = Self.platformLabel(for: activePlatforms)
        } catch {
            messagingPlatformLabel = "No messaging connected"
        }
    }

    private static let messagingChannels: [(key: String, name: String)] = [
        ("telegram", "Telegram"),
        ("discord", "Discord"),
        ("whatsapp", "WhatsApp"),
        ("slack", "Slack"),
        ("matrix", "Matrix"),
        ("line", "LINE"),
        ("feishu", "Feishu"),
        ("dingtalk", "DingTalk"),
        ("qq", "QQ"),
        ("onebot", "OneBot"),
        ("wecom", "WeCom"),
        ("wecom_app", "WeCom"),
        ("wecom_aibot", "WeCom"),
        ("weixin", "WeChat"),
        ("irc", "IRC"),
        ("maixcam", "MaixCam"),
    ]

    private static func platformLabel(for platforms: [String]) -> String {
        let unique = platforms.reduce(into: [String]()) { result, platform in
            if !result.contains(platform) { result.append(platform) }
        }
        guard !unique.isEmpty else { return "No messaging connected" }
        return "\(unique.joined(separator: " + ")) connected"
    }
}

struct JameRootView: View {
    @StateObject private var settings: LauncherSettingsStore
    @StateObject private var chat: NativeChatStore
    @StateObject private var navigationContext: NativeDesktopNavigationStore
    @State private var showingCommandPalette = false
    @State private var showingTerminalWorkspace = false
    @AppStorage("launcher.design.theme") private var savedTheme = LauncherTheme.light.rawValue
    @AppStorage("launcher.design.accent") private var savedAccent = LauncherAccent.theme.rawValue
    @AppStorage("launcher.design.windowOpacity") private var windowOpacity = 1.0
    @Environment(\.colorScheme) private var systemColorScheme
    // The native launcher is a chat app first. Keep it selected on launch,
    // while retaining a persistent, desktop-native list of supporting views.
    @Binding var selectedSection: DesktopSection?

    init(selectedSection: Binding<DesktopSection?>) {
        let settings = LauncherSettingsStore()
        let port = Int(settings.port) ?? 18800
        _settings = StateObject(wrappedValue: settings)
        _chat = StateObject(wrappedValue: NativeChatStore(port: port))
        _navigationContext = StateObject(wrappedValue: NativeDesktopNavigationStore(port: port))
        _selectedSection = selectedSection
    }

    private var chromeTheme: LauncherTheme {
        launcherThemePreference(from: savedTheme).resolved(for: systemColorScheme)
    }
    private var chromeBackground: Color { chromeTheme == .light ? Color(red: 0.965, green: 0.965, blue: 0.955) : JameBrand.ink }
    private var chromePanel: Color { chromeTheme == .light ? .white : JameBrand.elevated }
    private var chromeRule: Color { chromeTheme == .light ? Color.black.opacity(0.14) : JameBrand.rule }
    private var chromeForeground: Color { chromeTheme == .light ? JameBrand.ink : JameBrand.paper }
    private var chromeAccent: Color { launcherAccentPreference(from: savedAccent, theme: chromeTheme) }
    private var detailBackgroundGradient: LinearGradient {
        LinearGradient(
            colors: [chromeBackground, chromeAccent.opacity(chromeTheme == .light ? 0.045 : 0.075), chromeBackground],
            startPoint: .topLeading,
            endPoint: .bottomTrailing
        )
    }
    private var stableSectionSelection: Binding<DesktopSection?> {
        Binding(
            get: { selectedSection },
            set: { selectDesktopSection($0, selection: $selectedSection) }
        )
    }

    @ViewBuilder
    private var detailContent: some View {
        switch selectedSection ?? .chat {
        case .chat:
            ChatView(port: Int(settings.port) ?? 18800, chat: chat)
        case .fixedChats:
            SessionsView(port: Int(settings.port) ?? 18800, pinnedOnly: true, resumeSession: openSessionInChat)
        case .memory:
            AgentMemoryView(port: Int(settings.port) ?? 18800)
        case .agent:
            AgentManagerView(port: Int(settings.port) ?? 18800)
        case .sessions:
            SessionsView(port: Int(settings.port) ?? 18800, resumeSession: openSessionInChat)
        case .archivedChats:
            SessionsView(port: Int(settings.port) ?? 18800, archivedOnly: true, resumeSession: openSessionInChat)
        case .automations:
            AutomationsView(port: Int(settings.port) ?? 18800)
        case .capabilities:
            CapabilitiesView(port: Int(settings.port) ?? 18800)
        case .artifacts:
            ArtifactsView()
        case .settings:
            QuickSettingsView(settings: settings) {
                selectDesktopSection(.archivedChats, selection: $selectedSection)
            }
        }
    }

    var body: some View {
        NavigationSplitView {
            DesktopSidebar(
                selectedSection: stableSectionSelection,
                isAgentWorking: chat.isResponseInProgress,
                hasAutomations: navigationContext.hasAutomations,
                hasFixedChats: navigationContext.hasFixedChats,
                messagingPlatformLabel: navigationContext.messagingPlatformLabel,
                showingCommandPalette: $showingCommandPalette,
                openTerminal: { showingTerminalWorkspace = true }
            )
        } detail: {
            ZStack {
                detailBackgroundGradient
                .ignoresSafeArea()
                detailContent
            }
        }
        .tint(chromeAccent)
        .buttonBorderShape(.roundedRectangle(radius: 0))
        .textFieldStyle(SquareTextFieldStyle())
        .preferredColorScheme(launcherThemePreference(from: savedTheme).preferredColorScheme)
        .background {
            ZStack {
                WindowOpacityConfigurator(opacity: windowOpacity)
                WindowBehaviorConfigurator()
            }
            .frame(width: 0, height: 0)
        }
        .navigationSplitViewStyle(.balanced)
        .navigationSplitViewColumnWidth(min: 170, ideal: 220, max: 290)
        .onReceive(
            DistributedNotificationCenter.default().publisher(for: .jameclawHomeNavigation)
        ) { notification in
            guard let sectionName = notification.userInfo?["section"] as? String else { return }
            let section = (sectionName == "skills" || sectionName == "connectors")
                ? DesktopSection.capabilities
                : DesktopSection(rawValue: sectionName)
            guard let section else { return }
            selectDesktopSection(section, selection: $selectedSection)
            if notification.userInfo?["new_chat"] as? Bool == true {
                // Defer until the chat view is mounted when the launcher has
                // just opened Jame, while immediately clearing an existing chat.
                DispatchQueue.main.async {
                    NotificationCenter.default.post(name: .jameclawNewChat, object: nil)
                }
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: .jameclawCommandPalette)) { _ in
            showingCommandPalette = true
        }
        .onReceive(NotificationCenter.default.publisher(for: .jameclawAutomationChanged)) { _ in
            Task { await navigationContext.refresh() }
        }
        .onReceive(NotificationCenter.default.publisher(for: .jameclawFixedChatsChanged)) { _ in
            Task { await navigationContext.refresh() }
        }
        .task {
            await navigationContext.monitor()
        }
        .sheet(isPresented: $showingTerminalWorkspace) {
            TerminalWorkspaceView(
                port: Int(settings.port) ?? 18800,
                isPresented: $showingTerminalWorkspace
            )
        }
        // Do not impose an application-level minimum or fixed content size.
        // This lets people size the Jame window however they prefer, including
        // narrow and compact layouts managed by macOS.
    }

    private func openSessionInChat(_ request: NativeSessionResumeRequest) {
        selectDesktopSection(.chat, selection: $selectedSection)
        // The session list and Chat are mutually exclusive detail views. Post
        // after SwiftUI mounts Chat so its store receives the handoff.
        DispatchQueue.main.async {
            NotificationCenter.default.post(name: .jameclawResumeSession, object: request)
        }
    }
}

private struct DesktopSidebar: View {
    @Binding var selectedSection: DesktopSection?
    let isAgentWorking: Bool
    let hasAutomations: Bool
    let hasFixedChats: Bool
    let messagingPlatformLabel: String
    @Binding var showingCommandPalette: Bool
    let openTerminal: () -> Void
    @AppStorage("launcher.design.theme") private var savedTheme = LauncherTheme.light.rawValue
    @AppStorage("launcher.design.accent") private var savedAccent = LauncherAccent.theme.rawValue
    @Environment(\.colorScheme) private var colorScheme
    @State private var sessionsGlow = false

    private var background: Color { colorScheme == .light ? Color(red: 0.965, green: 0.965, blue: 0.955) : JameBrand.ink }
    private var panel: Color { colorScheme == .light ? .white : JameBrand.elevated }
    private var primary: Color { colorScheme == .light ? JameBrand.ink : JameBrand.paper }
    private var muted: Color { colorScheme == .light ? JameBrand.ink.opacity(0.58) : JameBrand.muted }
    private var rule: Color { colorScheme == .light ? JameBrand.ink.opacity(0.14) : JameBrand.rule }
    private var theme: LauncherTheme { launcherThemePreference(from: savedTheme).resolved(for: colorScheme) }
    private var accent: Color { launcherAccentPreference(from: savedAccent, theme: theme) }

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 2) {
                DesktopNavigationHeader(title: "Workspace")
                    .padding(.horizontal, 14)
                ForEach(DesktopSection.workspace.filter { $0 != .fixedChats || hasFixedChats }) { section in
                    navigationButton(section)
                }
                DesktopNavigationHeader(title: "Agent")
                    .padding(.horizontal, 14)
                    .padding(.top, 8)
                ForEach(DesktopSection.agentTools.filter { $0 != .automations || hasAutomations }) { section in
                    navigationButton(section)
                }
            }
            .padding(.vertical, 8)
        }
        .background(background)
        .safeAreaInset(edge: .top, spacing: 0) {
            VStack(alignment: .leading, spacing: 10) {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        HStack(spacing: 0) {
                            Text("JameClaw").foregroundStyle(primary)
                            Text(".").foregroundStyle(accent)
                        }
                        .font(.system(.headline, design: .rounded).weight(.semibold))
                        Label("Local agent", systemImage: "checkmark.circle.fill")
                            .font(.caption)
                            .foregroundStyle(muted)
                            .symbolRenderingMode(.hierarchical)
                    }
                    Spacer()
                    Button(action: openTerminal) {
                        Image(systemName: "terminal")
                            .frame(width: 28, height: 24)
                            .foregroundStyle(primary)
                            .background(panel)
                            .overlay(Rectangle().stroke(rule))
                            .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .help("Open terminal and choose where Jame works")
                    Button {
                        selectedSection = .settings
                    } label: {
                        Image(systemName: selectedSection == .settings ? "gearshape.fill" : "gearshape")
                            .frame(width: 28, height: 24)
                            .foregroundStyle(selectedSection == .settings ? JameBrand.ink : primary)
                            .background(selectedSection == .settings ? accent : panel)
                            .overlay(Rectangle().stroke(rule))
                            .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .help("Open JameClaw Settings")
                }
                Button {
                    showingCommandPalette = true
                } label: {
                    HStack(spacing: 7) {
                        Image(systemName: "magnifyingglass")
                        Text("Quick actions")
                        Spacer()
                        Text("⌘K").font(.caption.monospaced()).foregroundStyle(.tertiary)
                    }
                    .font(.caption.weight(.medium))
                    .padding(.horizontal, 9).padding(.vertical, 7)
                    .foregroundStyle(primary)
                    .background(panel, in: Rectangle())
                    .overlay(Rectangle().stroke(rule))
                }
                .buttonStyle(.plain)
                .popover(
                    isPresented: $showingCommandPalette,
                    attachmentAnchor: .rect(.bounds),
                    arrowEdge: .top
                ) {
                    QuickActionPalette(selectedSection: $selectedSection) {
                        showingCommandPalette = false
                    }
                }
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 14)
            .background(background.opacity(0.97))
        }
        .safeAreaInset(edge: .bottom, spacing: 0) {
            Button {
                selectedSection = .settings
            } label: {
                HStack(spacing: 7) {
                    Circle().fill(accent).frame(width: 7, height: 7)
                    Text(messagingPlatformLabel)
                        .font(.caption.weight(.medium))
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                    Spacer()
                    Image(systemName: "chevron.right")
                        .font(.caption2.weight(.bold))
                        .foregroundStyle(.tertiary)
                }
            }
            .buttonStyle(.plain)
            .help("Open Settings to manage connected platforms")
            .padding(.horizontal, 16)
            .padding(.vertical, 11)
            .background(background.opacity(0.97))
        }
        .navigationTitle("")
        .onAppear { sessionsGlow = isAgentWorking }
        .onChange(of: isAgentWorking) { _, working in
            sessionsGlow = working
        }
    }

    private func navigationButton(_ section: DesktopSection) -> some View {
        let showWorkingGlow = isAgentWorking && selectedSection != .chat && section == .sessions
        return Button {
            selectedSection = section
        } label: {
            DesktopNavigationRow(section: section, accent: accent, isWorking: showWorkingGlow)
                .padding(.horizontal, 14)
                .padding(.vertical, 6)
                .frame(maxWidth: .infinity, alignment: .leading)
                .contentShape(Rectangle())
                .foregroundStyle(primary)
                .background(
                    selectedSection == section
                        ? panel
                        : showWorkingGlow ? accent.opacity(sessionsGlow ? 0.22 : 0.08) : Color.clear
                )
                .overlay(alignment: .leading) {
                    if selectedSection == section {
                        Rectangle().fill(accent).frame(width: 3)
                    }
                }
                .overlay {
                    if showWorkingGlow {
                        Rectangle()
                            .stroke(accent.opacity(sessionsGlow ? 0.95 : 0.36), lineWidth: sessionsGlow ? 1.7 : 1)
                    }
                }
                .shadow(
                    color: showWorkingGlow ? accent.opacity(sessionsGlow ? 0.76 : 0.20) : .clear,
                    radius: showWorkingGlow ? (sessionsGlow ? 11 : 3) : 0
                )
                .scaleEffect(showWorkingGlow && sessionsGlow ? 1.012 : 1)
                .animation(
                    showWorkingGlow
                        ? .easeInOut(duration: 0.72).repeatForever(autoreverses: true)
                        : .easeOut(duration: 0.18),
                    value: sessionsGlow
                )
        }
        .buttonStyle(.plain)
        .accessibilityLabel(
            showWorkingGlow
                ? "Sessions — Jame is still working in Chat"
                : section.title
        )
        .help(showWorkingGlow ? "Jame is still working. Open Sessions or return to Chat." : section.title)
    }
}

private struct QuickActionPalette: View {
    @Binding var selectedSection: DesktopSection?
    let dismiss: () -> Void
    @State private var search = ""
    @FocusState private var isSearchFocused: Bool

    private struct Action: Identifiable {
        let id: String
        let title: String
        let detail: String
        let symbol: String
        let section: DesktopSection
        let createsChat: Bool
    }

    private let actions: [Action] = [
        Action(id: "new-chat", title: "Start a new chat", detail: "Ask JameClaw for help", symbol: "square.and.pencil", section: .chat, createsChat: true),
        Action(id: "chat", title: "Open chat", detail: "Continue a conversation", symbol: "message.fill", section: .chat, createsChat: false),
        Action(id: "agents", title: "Manage agents", detail: "Team agents and subagents", symbol: "sparkles", section: .agent, createsChat: false),
        Action(id: "mcp", title: "Open capabilities", detail: "Manage skills and MCP servers", symbol: "wand.and.stars.inverse", section: .capabilities, createsChat: false),
        Action(id: "memory", title: "Review memory", detail: "See and edit what JameClaw remembers", symbol: "brain.head.profile", section: .memory, createsChat: false),
        Action(id: "automation", title: "Open automations", detail: "Create and manage scheduled work", symbol: "calendar.badge.clock", section: .automations, createsChat: false),
        Action(id: "sessions", title: "Browse sessions", detail: "Find a past conversation", symbol: "clock.arrow.circlepath", section: .sessions, createsChat: false),
        Action(id: "settings", title: "Open settings", detail: "Configure JameClaw Desktop", symbol: "gearshape", section: .settings, createsChat: false),
    ]

    private var filteredActions: [Action] {
        let query = search.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !query.isEmpty else { return actions }
        return actions.filter { $0.title.lowercased().contains(query) || $0.detail.lowercased().contains(query) }
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 10) {
                Image(systemName: "command").foregroundStyle(.tint)
                TextField("Search actions…", text: $search)
                    .textFieldStyle(.plain)
                    .font(.title3)
                    .focused($isSearchFocused)
                Text("⌘K").font(.caption.monospaced()).foregroundStyle(.secondary)
            }
            .padding(16)
            Divider()
            ScrollView {
                LazyVStack(spacing: 4) {
                    ForEach(filteredActions) { action in
                        Button { perform(action) } label: {
                            HStack(spacing: 12) {
                                Image(systemName: action.symbol)
                                    .foregroundStyle(action.section.tint)
                                    .frame(width: 22)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(action.title).font(.body.weight(.semibold))
                                    Text(action.detail).font(.caption).foregroundStyle(.secondary)
                                }
                                Spacer()
                                Image(systemName: "return").font(.caption).foregroundStyle(.tertiary)
                            }
                            .padding(.horizontal, 14).padding(.vertical, 10)
                            .contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(8)
            }
            .frame(height: 360)
            if filteredActions.isEmpty {
                Text("No matching actions").font(.caption).foregroundStyle(.secondary).padding(.bottom, 18)
            }
        }
        .frame(width: 500)
        .onAppear { isSearchFocused = true }
    }

    private func perform(_ action: Action) {
        selectedSection = action.section
        if action.createsChat {
            DispatchQueue.main.async { NotificationCenter.default.post(name: .jameclawNewChat, object: nil) }
        }
        dismiss()
    }
}

private struct DesktopNavigationHeader: View {
    let title: String

    var body: some View {
        Text(title.uppercased())
            .font(.system(size: 10, weight: .semibold, design: .rounded))
            .foregroundStyle(.tertiary)
            .tracking(0.8)
            .padding(.top, 7)
    }
}

private struct DesktopNavigationRow: View {
    let section: DesktopSection
    let accent: Color
    let isWorking: Bool

    var body: some View {
        HStack(spacing: 8) {
            Label {
                Text(section.title).font(.system(.body, design: .rounded).weight(.medium))
            } icon: {
                Image(systemName: section.symbol)
                    .symbolRenderingMode(.hierarchical)
                    .foregroundStyle(accent)
                    .frame(width: 18)
            }
            Spacer(minLength: 4)
            if isWorking {
                Text("...")
                    .font(.system(size: 11, weight: .bold, design: .rounded))
                    .foregroundStyle(accent)
                    .accessibilityHidden(true)
            }
        }
        .padding(.vertical, 2)
    }
}

enum DesktopSection: String, CaseIterable, Identifiable {
    case chat
    case fixedChats
    case memory
    case sessions
    case archivedChats
    case agent
    case artifacts
    case capabilities
    case automations
    case settings

    var id: Self { self }
    var title: String {
        switch self {
        case .chat: return "Chat"
        case .fixedChats: return "Fixed Chats"
        case .memory: return "Memory"
        case .sessions: return "Sessions"
        case .archivedChats: return "Archived Chats"
        case .agent: return "Agent"
        case .artifacts: return "Artifacts"
        case .capabilities: return "Capabilities"
        case .automations: return "Automations"
        case .settings: return "Settings"
        }
    }

    static let workspace: [DesktopSection] = [.chat, .fixedChats, .sessions, .artifacts]
    static let agentTools: [DesktopSection] = [.agent, .memory, .capabilities, .automations]

    var tint: Color {
        JameBrand.orange
    }
    var symbol: String {
        switch self {
        case .chat: return "message.fill"
        case .fixedChats: return "pin.fill"
        case .memory: return "brain.head.profile"
        case .agent: return "sparkles"
        case .sessions: return "clock.arrow.circlepath"
        case .archivedChats: return "archivebox.fill"
        case .automations: return "calendar.badge.clock"
        case .capabilities: return "wand.and.stars.inverse"
        case .artifacts: return "shippingbox.fill"
        case .settings: return "gearshape"
        }
    }

    var menuShortcut: KeyEquivalent {
        switch self {
        case .chat: return "1"
        case .fixedChats: return "2"
        case .memory: return "3"
        case .agent: return "2"
        case .artifacts: return "3"
        case .capabilities: return "4"
        case .sessions: return "5"
        case .archivedChats: return "8"
        case .automations: return "6"
        case .settings: return ","
        }
    }
}

private struct NativeAutomationResponse: Codable {
    let items: [NativeAutomation]
}

private struct NativeAutomationBlueprintResponse: Codable {
    let blueprints: [NativeAutomationBlueprint]
}

private struct NativeAutomationBlueprint: Codable, Identifiable {
    let key: String
    let title: String
    let description: String
    let fields: [NativeAutomationBlueprintField]
    let tags: [String]

    var id: String { key }
}

private struct NativeAutomationBlueprintField: Codable, Identifiable {
    let name: String
    let type: String
    let label: String
    let defaultValue: String?
    let options: [String]?
    let help: String?

    var id: String { name }

    enum CodingKeys: String, CodingKey {
        case name, type, label, options, help
        case defaultValue = "default"
    }
}

private struct NativeAutomationBlueprintRequest: Encodable {
    let blueprint: String
    let values: [String: String]
}

private struct NativeAutomation: Codable, Identifiable {
    let id: String
    let name: String
    let enabled: Bool
    let status: String
    let schedule: String
    let prompt: String
    let delivery: String
    let nextRunAtMS: Int64?
    let lastRunAtMS: Int64?
    let lastStatus: String?
    let lastError: String?
    let running: Bool
    let createdAtMS: Int64

    enum CodingKeys: String, CodingKey {
        case id, name, enabled, status, schedule, prompt, delivery, running
        case nextRunAtMS = "next_run_at_ms"
        case lastRunAtMS = "last_run_at_ms"
        case lastStatus = "last_status"
        case lastError = "last_error"
        case createdAtMS = "created_at_ms"
    }
}

private struct NativeAutomationOutput: Codable {
    let automationID: String
    let status: String?
    let ranAtMS: Int64?
    let content: String

    enum CodingKeys: String, CodingKey {
        case status, content
        case automationID = "automation_id"
        case ranAtMS = "ran_at_ms"
    }
}

private func nativeAutomationDate(_ timestamp: Int64?) -> String {
    guard let timestamp, timestamp > 0 else { return "Not scheduled" }
    return Date(timeIntervalSince1970: TimeInterval(timestamp) / 1000)
        .formatted(date: .abbreviated, time: .shortened)
}

@MainActor
private final class NativeAutomationStore: ObservableObject {
    @Published var automations: [NativeAutomation] = []
    @Published var blueprints: [NativeAutomationBlueprint] = []
    @Published var isLoading = false
    @Published var error = ""
    @Published var runningID: String?
    @Published var output: NativeAutomationOutput?
    @Published var outputID: String?
    @Published var outputError = ""
    @Published var schedulingBlueprintKey: String?

    private let port: Int

    init(port: Int) { self.port = port }

    func load() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let (data, response) = try await URLSession.shared.data(
                for: authenticatedConsoleRequest(port: port, path: "/api/automation")
            )
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            automations = try JSONDecoder().decode(NativeAutomationResponse.self, from: data).items
            error = ""
        } catch {
            self.error = "Could not load automations. Start JameClaw and try again."
        }

        do {
            let (data, response) = try await URLSession.shared.data(
                for: authenticatedConsoleRequest(port: port, path: "/api/automation/blueprints")
            )
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            blueprints = try JSONDecoder().decode(NativeAutomationBlueprintResponse.self, from: data).blueprints
        } catch {
            // Existing scheduled automations remain useful even when blueprints
            // are temporarily unavailable, so keep this failure non-blocking.
            if automations.isEmpty { self.error = "Could not load automation templates. Start JameClaw and try again." }
        }
    }

    func schedule(_ blueprint: NativeAutomationBlueprint, values: [String: String]) async -> Bool {
        schedulingBlueprintKey = blueprint.key
        defer { schedulingBlueprintKey = nil }
        do {
            var request = authenticatedConsoleRequest(
                port: port,
                path: "/api/automation/blueprints/instantiate",
                method: "POST"
            )
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONEncoder().encode(
                NativeAutomationBlueprintRequest(blueprint: blueprint.key, values: values)
            )
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            await load()
            NotificationCenter.default.post(name: .jameclawAutomationChanged, object: nil)
            return true
        } catch {
            self.error = "Could not schedule \(blueprint.title). Check the template fields and try again."
            return false
        }
    }

    func run(_ automation: NativeAutomation) async {
        runningID = automation.id
        defer { runningID = nil }
        do {
            let (_, response) = try await URLSession.shared.data(
                for: authenticatedConsoleRequest(
                    port: port,
                    path: "/api/automation/\(automation.id.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? automation.id)/run",
                    method: "POST"
                )
            )
            guard let http = response as? HTTPURLResponse, http.statusCode == 202 else {
                throw URLError(.badServerResponse)
            }
            await load()
        } catch {
            self.error = "Could not start \(automation.name). Make sure the gateway is running."
        }
    }

    func loadOutput(for automation: NativeAutomation) async {
        outputID = automation.id
        output = nil
        outputError = ""
        do {
            let (data, response) = try await URLSession.shared.data(
                for: authenticatedConsoleRequest(
                    port: port,
                    path: "/api/automation/\(automation.id.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? automation.id)/output"
                )
            )
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            output = try JSONDecoder().decode(NativeAutomationOutput.self, from: data)
        } catch {
            outputError = "No saved output is available for this automation yet."
        }
    }
}

private struct AutomationsView: View {
    @StateObject private var store: NativeAutomationStore
    @State private var showingTemplateGallery = false
    @State private var blueprintToConfigure: NativeAutomationBlueprint?
    @State private var selectedBlueprint: NativeAutomationBlueprint?

    init(port: Int) { _store = StateObject(wrappedValue: NativeAutomationStore(port: port)) }

    var body: some View {
        VStack(spacing: 0) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 3) {
                    Text("Automations").font(.title2.weight(.semibold))
                    Text("Scheduled work shared with the JameClaw Web Console")
                        .font(.caption).foregroundStyle(.secondary)
                }
                Spacer()
                Button {
                    showingTemplateGallery = true
                } label: {
                    Label("New automation", systemImage: "plus")
                }
                .buttonStyle(.borderedProminent)
                Button { Task { await store.load() } } label: {
                    Image(systemName: "arrow.clockwise")
                }
                .help("Refresh automations")
            }
            .padding(18)

            if store.isLoading && store.automations.isEmpty && store.blueprints.isEmpty {
                Spacer()
                ProgressView("Loading automations…")
                Spacer()
            } else {
                List {
                    Section {
                        if store.automations.isEmpty {
                            Text("No automations have been scheduled yet.")
                                .foregroundStyle(.secondary)
                        } else {
                            ForEach(store.automations) { automation in
                                AutomationRow(automation: automation, store: store)
                            }
                        }
                    } header: {
                        Text("Your scheduled automations")
                    }
                }
                .listStyle(.inset)
            }

            if !store.error.isEmpty && !store.automations.isEmpty {
                Text(store.error).font(.caption).foregroundStyle(.red)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 18).padding(.bottom, 10)
            }
        }
        .task { await store.load() }
        .sheet(isPresented: $showingTemplateGallery, onDismiss: {
            if let blueprintToConfigure {
                self.blueprintToConfigure = nil
                selectedBlueprint = blueprintToConfigure
            }
        }) {
            AutomationTemplateGallery(blueprints: store.blueprints) { blueprint in
                blueprintToConfigure = blueprint
                showingTemplateGallery = false
            }
        }
        .sheet(item: $selectedBlueprint) { blueprint in
            AutomationTemplateSetupView(blueprint: blueprint, store: store)
        }
    }
}

private struct AutomationTemplateGallery: View {
    let blueprints: [NativeAutomationBlueprint]
    let choose: (NativeAutomationBlueprint) -> Void
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                VStack(alignment: .leading, spacing: 3) {
                    Text("New automation").font(.title2.weight(.semibold))
                    Text("Choose a suggestion to customize and schedule it.")
                        .font(.caption).foregroundStyle(.secondary)
                }
                Spacer()
                Button("Done") { dismiss() }
            }
            .padding(20)
            Divider()
            if blueprints.isEmpty {
                ContentUnavailableView(
                    "No suggestions available",
                    systemImage: "sparkles",
                    description: Text("Start JameClaw and refresh this page to load automation templates.")
                )
            } else {
                List(blueprints) { blueprint in
                    AutomationTemplateRow(blueprint: blueprint) { choose(blueprint) }
                }
                .listStyle(.inset)
            }
        }
        .frame(minWidth: 560, minHeight: 460)
    }
}

private struct AutomationTemplateRow: View {
    let blueprint: NativeAutomationBlueprint
    let setUp: () -> Void

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            Image(systemName: "sparkles")
                .foregroundStyle(.tint)
                .frame(width: 22)
            VStack(alignment: .leading, spacing: 4) {
                Text(blueprint.title).font(.headline)
                Text(blueprint.description).font(.caption).foregroundStyle(.secondary)
                if !blueprint.tags.isEmpty {
                    Text(blueprint.tags.map { "#\($0)" }.joined(separator: "  "))
                        .font(.caption2).foregroundStyle(.secondary)
                }
            }
            Spacer()
            Button("Set up", action: setUp)
                .controlSize(.small)
        }
        .padding(.vertical, 5)
    }
}

private struct AutomationTemplateSetupView: View {
    let blueprint: NativeAutomationBlueprint
    @ObservedObject var store: NativeAutomationStore
    @Environment(\.dismiss) private var dismiss
    @State private var values: [String: String]

    init(blueprint: NativeAutomationBlueprint, store: NativeAutomationStore) {
        self.blueprint = blueprint
        self.store = store
        _values = State(initialValue: Dictionary(uniqueKeysWithValues: blueprint.fields.map { ($0.name, $0.defaultValue ?? "") }))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Set up \(blueprint.title)").font(.title2.weight(.semibold))
            Text(blueprint.description).foregroundStyle(.secondary)
            Form {
                ForEach(blueprint.fields) { field in
                    if let options = field.options, !options.isEmpty {
                        Picker(field.label, selection: valueBinding(for: field.name)) {
                            ForEach(options, id: \.self) { option in
                                Text(option).tag(option)
                            }
                        }
                    } else {
                        TextField(field.label, text: valueBinding(for: field.name))
                    }
                    if let help = field.help, !help.isEmpty {
                        Text(help).font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
            .formStyle(.grouped)
            if !store.error.isEmpty {
                Text(store.error).font(.caption).foregroundStyle(.red)
            }
            HStack {
                Spacer()
                Button("Cancel") { dismiss() }
                Button(store.schedulingBlueprintKey == blueprint.key ? "Scheduling…" : "Schedule") {
                    Task {
                        if await store.schedule(blueprint, values: values) { dismiss() }
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(store.schedulingBlueprintKey != nil)
            }
        }
        .padding(24)
        .frame(width: 520)
    }

    private func valueBinding(for fieldName: String) -> Binding<String> {
        Binding(
            get: { values[fieldName] ?? "" },
            set: { values[fieldName] = $0 }
        )
    }
}

private struct AutomationRow: View {
    let automation: NativeAutomation
    @ObservedObject var store: NativeAutomationStore

    private var status: String { automation.running ? "Running" : (automation.enabled ? automation.status.capitalized : "Disabled") }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 3) {
                    Text(automation.name).font(.headline)
                    Text(automation.schedule).font(.subheadline).foregroundStyle(.secondary)
                }
                Spacer()
                Text(status)
                    .font(.caption.weight(.medium))
                    .padding(.horizontal, 8).padding(.vertical, 4)
                    .background(status == "Error" ? Color.red.opacity(0.15) : Color.secondary.opacity(0.12), in: Rectangle())
                Button(store.runningID == automation.id || automation.running ? "Running…" : "Run now") {
                    Task { await store.run(automation) }
                }
                .disabled(!automation.enabled || automation.running || store.runningID != nil)
                .buttonStyle(.borderedProminent)
                .controlSize(.small)
            }

            Grid(alignment: .leading, horizontalSpacing: 22, verticalSpacing: 6) {
                GridRow {
                    Text("Request").foregroundStyle(.secondary)
                    Text(automation.prompt.isEmpty ? "No request recorded" : automation.prompt)
                }
                GridRow {
                    Text("Delivery").foregroundStyle(.secondary)
                    Text(automation.delivery)
                }
                GridRow {
                    Text("Next run").foregroundStyle(.secondary)
                    Text(nativeAutomationDate(automation.nextRunAtMS))
                }
                GridRow {
                    Text("Last result").foregroundStyle(.secondary)
                    Text(automation.lastStatus?.isEmpty == false ? automation.lastStatus! : "No result yet")
                }
            }
            .font(.caption)

            if let lastError = automation.lastError, !lastError.isEmpty {
                Label(lastError, systemImage: "exclamationmark.triangle.fill")
                    .font(.caption).foregroundStyle(.red)
            }

            HStack {
                Button("View last output") { Task { await store.loadOutput(for: automation) } }
                    .controlSize(.small)
                    .disabled(
                        automation.lastRunAtMS == nil ||
                        (store.outputID == automation.id && store.output == nil && store.outputError.isEmpty)
                    )
                if store.outputID == automation.id && store.output == nil && store.outputError.isEmpty {
                    ProgressView().controlSize(.small)
                }
                Spacer()
                if let lastRun = automation.lastRunAtMS {
                    Text("Last run \(nativeAutomationDate(lastRun))").font(.caption).foregroundStyle(.secondary)
                }
            }

            if store.outputID == automation.id {
                if let output = store.output {
                    Text(output.content.isEmpty ? "The run completed without output." : output.content)
                        .font(.system(.caption, design: .monospaced))
                        .textSelection(.enabled)
                        .lineLimit(12)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(10)
                        .background(Color.secondary.opacity(0.08), in: Rectangle())
                } else if !store.outputError.isEmpty {
                    Text(store.outputError).font(.caption).foregroundStyle(.red)
                }
            }
        }
        .padding(.vertical, 7)
    }
}

private struct NativeSessionSummary: Codable, Identifiable {
    let id: String
    let title: String
    let preview: String
    let messageCount: Int
    let updated: String
    let channel: String?
    let chatType: String?
    let chatID: String?
    let pinned: Bool
    let archived: Bool

    enum CodingKeys: String, CodingKey {
        case id, title, preview, updated, channel, pinned, archived
        case chatType = "chat_type"
        case chatID = "chat_id"
        case messageCount = "message_count"
    }
}

private struct NativeSessionMessage: Codable, Identifiable {
    let role: String
    let content: String
    let id = UUID()

    enum CodingKeys: String, CodingKey {
        case role, content
    }
}

private struct NativeSessionDetail: Codable {
    let id: String
    let messages: [NativeSessionMessage]
}

private struct NativeSessionResumeResponse: Codable {
    let sessionID: String
    let messages: [NativeSessionMessage]
    let cloned: Bool

    enum CodingKeys: String, CodingKey {
        case messages, cloned
        case sessionID = "session_id"
    }
}

private struct NativeSessionResumeRequest {
    let sessionID: String
    let title: String
    let messages: [NativeSessionMessage]
    let cloned: Bool
}

private func sessionSourceName(_ session: NativeSessionSummary) -> String {
    let channel = session.channel?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    // The desktop app and Web Console both use the Jame channel. Their
    // persisted sessions are intentionally shared, so show that clearly.
    if channel == "jame" { return "Jame (Desktop / Web)" }
    return channel.isEmpty ? "Terminal" : channel.capitalized
}

@MainActor
private final class NativeSessionStore: ObservableObject {
    @Published var sessions: [NativeSessionSummary] = []
    @Published var selectedSessionID: String?
    @Published var selectedSession: NativeSessionDetail?
    @Published var selectedSource = "All conversations"
    @Published var searchText = ""
    @Published var isLoading = false
    @Published var error = ""

    private let port: Int

    init(port: Int) { self.port = port }

    func load() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let (data, response) = try await URLSession.shared.data(
                from: authenticatedConsoleURL(
                    port: port,
                    path: "/api/sessions",
                    queryItems: [
                        URLQueryItem(name: "offset", value: "0"),
                        URLQueryItem(name: "limit", value: "10000"),
                    ]
                )
            )
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            sessions = try JSONDecoder().decode([NativeSessionSummary].self, from: data)
            self.error = sessions.isEmpty ? "No conversations have been saved yet." : ""
        } catch {
            self.error = "Could not load conversation history. Start JameClaw and try again."
        }
    }

    var sources: [String] {
        ["All conversations"] + Array(Set(sessions.map(sessionSourceName))).sorted()
    }

    var visibleSessions: [NativeSessionSummary] {
        sessions.filter { session in
            let matchesSource = selectedSource == "All conversations" || sessionSourceName(session) == selectedSource
            let query = searchText.trimmingCharacters(in: .whitespacesAndNewlines)
            let matchesQuery = query.isEmpty
                || session.title.localizedCaseInsensitiveContains(query)
                || session.preview.localizedCaseInsensitiveContains(query)
                || sessionSourceName(session).localizedCaseInsensitiveContains(query)
            return matchesSource && matchesQuery
        }
    }

    func select(_ id: String?) async {
        selectedSessionID = id
        selectedSession = nil
        guard let id else { return }
        do {
            let (data, response) = try await URLSession.shared.data(from: authenticatedSessionURL(port: port, id: id))
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            selectedSession = try JSONDecoder().decode(NativeSessionDetail.self, from: data)
            self.error = ""
        } catch {
            self.error = "Could not open this conversation."
        }
    }

    func setPinned(_ session: NativeSessionSummary, pinned: Bool) async {
        do {
            var request = authenticatedConsoleRequest(url: authenticatedSessionURL(port: port, id: session.id).appendingPathComponent("pin"), method: "PUT")
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: ["pinned": pinned])
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else { throw URLError(.badServerResponse) }
            await load()
            NotificationCenter.default.post(name: .jameclawFixedChatsChanged, object: nil)
        } catch { self.error = "Could not update fixed chat." }
    }

    func setArchived(_ session: NativeSessionSummary, archived: Bool) async {
        do {
            var request = authenticatedConsoleRequest(
                url: authenticatedSessionURL(port: port, id: session.id).appendingPathComponent("archive"),
                method: "PUT"
            )
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: ["archived": archived])
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            if selectedSessionID == session.id {
                selectedSessionID = nil
                selectedSession = nil
            }
            await load()
        } catch {
            self.error = archived ? "Could not archive this chat." : "Could not restore this chat."
        }
    }

    func rename(_ session: NativeSessionSummary, title: String) async -> Bool {
        let cleanTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !cleanTitle.isEmpty else {
            self.error = "Enter a session name."
            return false
        }
        do {
            var request = authenticatedConsoleRequest(
                url: authenticatedSessionURL(port: port, id: session.id).appendingPathComponent("title"),
                method: "PUT"
            )
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: ["title": cleanTitle])
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            await load()
            self.error = ""
            return true
        } catch {
            self.error = "Could not rename this session."
            return false
        }
    }

    func resume(_ session: NativeSessionSummary) async -> NativeSessionResumeRequest? {
        isLoading = true
        defer { isLoading = false }
        do {
            var request = authenticatedConsoleRequest(
                url: authenticatedSessionURL(port: port, id: session.id).appendingPathComponent("resume"),
                method: "POST"
            )
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            let (data, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            let resumed = try JSONDecoder().decode(NativeSessionResumeResponse.self, from: data)
            self.error = ""
            return NativeSessionResumeRequest(
                sessionID: resumed.sessionID,
                title: session.title.isEmpty ? session.preview : session.title,
                messages: resumed.messages,
                cloned: resumed.cloned
            )
        } catch {
            self.error = "Could not continue this conversation in Chat."
            return nil
        }
    }
}

private func sessionRelativeDate(_ raw: String) -> String {
    let precise = ISO8601DateFormatter()
    precise.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
    let standard = ISO8601DateFormatter()
    standard.formatOptions = [.withInternetDateTime]
    guard let date = precise.date(from: raw) ?? standard.date(from: raw) else { return raw }
    let relative = RelativeDateTimeFormatter()
    relative.unitsStyle = .full
    return relative.localizedString(for: date, relativeTo: Date())
}

private struct SessionsView: View {
    @StateObject private var store: NativeSessionStore
    let pinnedOnly: Bool
    let archivedOnly: Bool
    let resumeSession: (NativeSessionResumeRequest) -> Void
    @State private var sessionToRename: NativeSessionSummary?
    @State private var renameTitle = ""
    @State private var sessionScrollIndex = 0
    @Environment(\.colorScheme) private var colorScheme

    private var pageBackground: Color { colorScheme == .light ? Color(red: 0.965, green: 0.965, blue: 0.955) : JameBrand.ink }
    private var panel: Color { colorScheme == .light ? Color(red: 0.985, green: 0.985, blue: 0.975) : JameBrand.panel }
    private var elevated: Color { colorScheme == .light ? .white : JameBrand.elevated }
    private var primary: Color { colorScheme == .light ? JameBrand.ink : JameBrand.paper }
    private var muted: Color { colorScheme == .light ? JameBrand.ink.opacity(0.58) : JameBrand.muted }
    private var rule: Color { colorScheme == .light ? JameBrand.ink.opacity(0.14) : JameBrand.rule }

    init(
        port: Int,
        pinnedOnly: Bool = false,
        archivedOnly: Bool = false,
        resumeSession: @escaping (NativeSessionResumeRequest) -> Void
    ) {
        _store = StateObject(wrappedValue: NativeSessionStore(port: port))
        self.pinnedOnly = pinnedOnly
        self.archivedOnly = archivedOnly
        self.resumeSession = resumeSession
    }

    private var displayedSessions: [NativeSessionSummary] {
        store.visibleSessions.filter { session in
            if archivedOnly { return session.archived }
            return !session.archived && (!pinnedOnly || session.pinned)
        }
    }

    var body: some View {
        HStack(spacing: 0) {
            VStack(spacing: 0) {
                VStack(alignment: .leading, spacing: 14) {
                    HStack(alignment: .top) {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(archivedOnly ? "Archived Chats" : pinnedOnly ? "Fixed Chats" : "Sessions")
                                .font(.system(size: 25, weight: .semibold, design: .rounded))
                            Text(archivedOnly ? "Chats kept out of the active timeline." : pinnedOnly ? "The conversations worth keeping close." : "Every conversation, one searchable timeline.")
                                .font(.caption)
                                .foregroundStyle(muted)
                            Text("Double-click a session to continue it in Chat")
                                .font(.system(size: 10, weight: .semibold, design: .rounded))
                                .foregroundStyle(JameBrand.orange)
                            Text("All \(displayedSessions.count) matching sessions are loaded — scroll to see the complete list.")
                                .font(.system(size: 9, weight: .medium, design: .rounded))
                                .foregroundStyle(muted)
                        }
                        Spacer()
                        Text("\(displayedSessions.count)")
                            .font(.system(size: 22, weight: .semibold, design: .rounded))
                            .foregroundStyle(JameBrand.orange)
                    }

                    VStack(alignment: .leading, spacing: 9) {
                        HStack(spacing: 7) {
                            Image(systemName: "magnifyingglass").foregroundStyle(JameBrand.orange)
                            TextField("Search conversations", text: $store.searchText)
                                .textFieldStyle(.plain)
                        }
                        .padding(.horizontal, 10).padding(.vertical, 8)
                        .background(elevated, in: Rectangle())
                        .overlay(Rectangle().stroke(rule))

                        HStack {
                            JameDropdownPicker(
                                label: "Conversation source",
                                options: store.sources.map { (label: $0, value: $0) },
                                selection: $store.selectedSource,
                                minWidth: 158
                            )

                            Spacer()
                            Button { Task { await store.load() } } label: { Image(systemName: "arrow.clockwise") }
                                .buttonStyle(.bordered)
                                .help("Refresh session history")
                        }
                    }
                }
                .padding(20)

                Rectangle().fill(rule).frame(height: 1)

                ScrollViewReader { proxy in
                    VStack(spacing: 0) {
                        ScrollView {
                            LazyVStack(spacing: 0) {
                                ForEach(displayedSessions) { session in
                            HStack(alignment: .top, spacing: 11) {
                                HStack(alignment: .top, spacing: 11) {
                                    Circle()
                                        .fill(store.selectedSessionID == session.id ? JameBrand.ink : JameBrand.orange)
                                        .frame(width: 8, height: 8)
                                        .padding(.top, 6)
                                    VStack(alignment: .leading, spacing: 6) {
                                        Text(session.title.isEmpty ? session.preview : session.title)
                                            .font(.system(.body, design: .rounded).weight(.semibold))
                                            .foregroundStyle(store.selectedSessionID == session.id ? JameBrand.ink : primary)
                                            .lineLimit(2)
                                        HStack(spacing: 6) {
                                            Text(sessionSourceName(session).uppercased())
                                            Text("•")
                                            Text("\(session.messageCount) MSG")
                                            Spacer()
                                            Text(sessionRelativeDate(session.updated))
                                        }
                                        .font(.system(size: 9, weight: .semibold, design: .rounded))
                                        .foregroundStyle(store.selectedSessionID == session.id ? JameBrand.ink.opacity(0.65) : muted)
                                    }
                                    Spacer(minLength: 0)
                                }
                                .contentShape(Rectangle())
                                .gesture(
                                    TapGesture(count: 2)
                                        .exclusively(before: TapGesture(count: 1))
                                        .onEnded { gesture in
                                            switch gesture {
                                            case .first:
                                                Task {
                                                    guard let request = await store.resume(session) else { return }
                                                    resumeSession(request)
                                                }
                                            case .second:
                                                Task { await store.select(session.id) }
                                            }
                                        }
                                )
                                Menu {
                                    Button {
                                        renameTitle = session.title.isEmpty ? session.preview : session.title
                                        sessionToRename = session
                                    } label: {
                                        Label("Rename Session", systemImage: "pencil")
                                    }
                                    Divider()
                                    Button {
                                        Task { await store.setPinned(session, pinned: !session.pinned) }
                                    } label: {
                                        Label(
                                            session.pinned ? "Unfix Chat" : "Fix Chat",
                                            systemImage: session.pinned ? "pin.slash" : "pin"
                                        )
                                    }
                                    Divider()
                                    Button {
                                        Task { await store.setArchived(session, archived: !session.archived) }
                                    } label: {
                                        Label(
                                            session.archived ? "Restore to Sessions" : "Archive Chat",
                                            systemImage: session.archived ? "arrow.uturn.backward" : "archivebox"
                                        )
                                    }
                                } label: {
                                    Image(systemName: "ellipsis")
                                        .font(.system(size: 13, weight: .semibold))
                                        .foregroundStyle(store.selectedSessionID == session.id ? JameBrand.ink : JameBrand.orange)
                                        .frame(width: 28, height: 24)
                                        .background(store.selectedSessionID == session.id ? JameBrand.ink.opacity(0.08) : elevated)
                                        .overlay(Rectangle().stroke(store.selectedSessionID == session.id ? JameBrand.ink.opacity(0.18) : rule))
                                        .contentShape(Rectangle())
                                }
                                .menuStyle(.borderlessButton)
                                .menuIndicator(.hidden)
                                .fixedSize()
                                .help("Session actions")
                                .accessibilityLabel("Session actions")
                            }
                            .padding(.horizontal, 16).padding(.vertical, 13)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(store.selectedSessionID == session.id ? JameBrand.orange : Color.clear)
                            .contentShape(Rectangle())
                            .id(session.id)
                            Rectangle().fill(rule).frame(height: 1).padding(.leading, 35)
                        }
                    }
                        }
                        .scrollIndicators(.visible)
                        .overlay {
                            if store.isLoading { ProgressView().tint(JameBrand.orange) }
                            else if displayedSessions.isEmpty {
                                ContentUnavailableView(
                                    archivedOnly ? "No archived chats" : "No conversations",
                                    systemImage: archivedOnly ? "archivebox" : "clock",
                                    description: Text(store.error.isEmpty ? "Try another search or source." : store.error)
                                )
                            }
                        }
                        .frame(maxHeight: .infinity)

                        HStack(spacing: 10) {
                            Button {
                                jumpSessions(by: -10, proxy: proxy)
                            } label: {
                                Label("Newer 10", systemImage: "arrow.up")
                            }
                            .disabled(sessionScrollIndex == 0 || displayedSessions.isEmpty)
                            Spacer()
                            Text("\(displayedSessions.count) sessions loaded")
                                .font(.caption2.monospaced())
                                .foregroundStyle(muted)
                            Spacer()
                            Button {
                                jumpSessions(by: 10, proxy: proxy)
                            } label: {
                                Label("Older 10", systemImage: "arrow.down")
                            }
                            .disabled(sessionScrollIndex + 10 >= displayedSessions.count)
                        }
                        .buttonStyle(.bordered)
                        .controlSize(.small)
                        .padding(.horizontal, 12).padding(.vertical, 8)
                        .background(panel)
                    }
                }
            }
            .frame(minWidth: 230, idealWidth: 330, maxWidth: 430, maxHeight: .infinity)
            .background(panel)

            Rectangle().fill(rule).frame(width: 1)

            Group {
                if let session = store.selectedSession {
                    VStack(spacing: 0) {
                        HStack {
                            VStack(alignment: .leading, spacing: 3) {
                                Text("CONVERSATION")
                                    .font(.system(size: 10, weight: .semibold, design: .rounded))
                                    .tracking(1.2).foregroundStyle(JameBrand.orange)
                                Text("\(session.messages.count) saved messages")
                                    .font(.caption).foregroundStyle(muted)
                            }
                            Spacer()
                        }
                        .padding(20)
                        Rectangle().fill(rule).frame(height: 1)

                        ScrollView {
                            LazyVStack(spacing: 15) {
                                ForEach(session.messages) { message in
                                    let isUser = message.role == "user"
                                    VStack(alignment: isUser ? .trailing : .leading, spacing: 6) {
                                        Text(isUser ? "YOU" : "JAME")
                                            .font(.system(size: 9, weight: .semibold, design: .rounded))
                                            .tracking(1)
                                            .foregroundStyle(isUser ? JameBrand.orange : muted)
                                        Text(message.content.isEmpty ? "(no text content)" : message.content)
                                            .foregroundStyle(isUser ? JameBrand.ink : primary)
                                            .textSelection(.enabled)
                                            .padding(.horizontal, 14).padding(.vertical, 11)
                                            .background(isUser ? JameBrand.orange : elevated, in: Rectangle())
                                    }
                                    .frame(maxWidth: .infinity, alignment: isUser ? .trailing : .leading)
                                }
                            }
                            .padding(22)
                        }
                    }
                } else {
                    ContentUnavailableView(
                        "Select a session",
                        systemImage: "bubble.left.and.bubble.right",
                        description: Text("Choose a conversation to read its full history.")
                    )
                }
            }
            .frame(minWidth: 260, maxWidth: .infinity, maxHeight: .infinity)
            .background(pageBackground)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .foregroundStyle(primary)
        .task { await store.load() }
        .sheet(item: $sessionToRename) { session in
            VStack(alignment: .leading, spacing: 16) {
                Text("Rename Session").font(.title2.weight(.semibold))
                Text("Give this conversation a clear name. The original messages are not changed.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                TextField("Session name", text: $renameTitle)
                    .onSubmit { submitRename(session) }
                HStack {
                    Spacer()
                    Button("Cancel") { sessionToRename = nil }
                    Button("Rename") { submitRename(session) }
                        .buttonStyle(.borderedProminent)
                        .disabled(renameTitle.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
            .padding(24)
            .frame(width: 430)
        }
    }

    private func submitRename(_ session: NativeSessionSummary) {
        Task {
            if await store.rename(session, title: renameTitle) {
                sessionToRename = nil
            }
        }
    }

    private func jumpSessions(by amount: Int, proxy: ScrollViewProxy) {
        guard !displayedSessions.isEmpty else { return }
        sessionScrollIndex = min(max(sessionScrollIndex + amount, 0), displayedSessions.count - 1)
        withAnimation(.easeOut(duration: 0.2)) {
            proxy.scrollTo(displayedSessions[sessionScrollIndex].id, anchor: .top)
        }
    }

}

private struct MCPServerList: Codable {
    let enabled: Bool
    let servers: [MCPServer]
}

private struct MCPServer: Codable, Identifiable {
    let name: String
    let enabled: Bool
    let transport: String
    let command: String?
    let args: [String]?
    let url: String?
    var id: String { name }

    var endpoint: String {
        if transport == "stdio" {
            return ([command].compactMap { $0 } + (args ?? []))
                .filter { !$0.isEmpty }
                .joined(separator: " ")
        }
        return url ?? "No endpoint configured"
    }
}

private struct NativeModelList: Codable {
    let models: [NativeConnectorModel]
}

private struct NativeConnectorModel: Codable, Identifiable {
    let index: Int
    let modelName: String
    let model: String
    let connectMode: String?
    let workspace: String?
    let configured: Bool
    let isDefault: Bool

    enum CodingKeys: String, CodingKey {
        case index
        case modelName = "model_name"
        case model
        case connectMode = "connect_mode"
        case workspace
        case configured
        case isDefault = "is_default"
    }

    var id: Int { index }
    var isCLI: Bool {
        let protocolName = model.lowercased().split(separator: "/").first ?? ""
        return protocolName.contains("cli") || protocolName == "github-copilot" || connectMode == "stdio" || connectMode == "grpc"
    }
}

@MainActor
private final class ConnectorsStore: ObservableObject {
    @Published var mcpEnabled = false
    @Published var mcpServers: [MCPServer] = []
    @Published var cliModels: [NativeConnectorModel] = []
    @Published var isLoading = false
    @Published var error = ""

    private let port: Int
    init(port: Int) { self.port = port }

    func load() async {
        isLoading = true
        defer { isLoading = false }
        do {
            async let mcpRequest = URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: "/api/tools/mcp/servers"))
            async let modelsRequest = URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: "/api/models"))
            let (mcpData, mcpResponse) = try await mcpRequest
            let (modelData, modelResponse) = try await modelsRequest
            guard let mcpHTTP = mcpResponse as? HTTPURLResponse, (200..<300).contains(mcpHTTP.statusCode),
                  let modelHTTP = modelResponse as? HTTPURLResponse, (200..<300).contains(modelHTTP.statusCode) else {
                throw URLError(.badServerResponse)
            }
            let decoder = JSONDecoder()
            let mcp = try decoder.decode(MCPServerList.self, from: mcpData)
            let models = try decoder.decode(NativeModelList.self, from: modelData)
            mcpEnabled = mcp.enabled
            mcpServers = mcp.servers
            cliModels = models.models.filter(\.isCLI)
            self.error = ""
        } catch {
            self.error = "Could not load connectors. Check that JameClaw is running."
        }
    }

    func addMCPServer(name: String, apiKey: String, url: String) async -> Bool {
        let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedURL = url.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedName.isEmpty else {
            error = "Enter a server name."
            return false
        }
        guard let endpoint = URL(string: trimmedURL), let scheme = endpoint.scheme?.lowercased(), scheme == "https" || scheme == "http" else {
            error = "Enter a valid HTTP or HTTPS MCP link."
            return false
        }
        do {
            var request = authenticatedConsoleRequest(port: port, path: "/api/tools/mcp/servers", method: "POST")
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: [
                "name": trimmedName,
                "api_key": apiKey.trimmingCharacters(in: .whitespacesAndNewlines),
                "url": trimmedURL,
                "transport": "http",
                "enabled": true,
            ])
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else { throw URLError(.badServerResponse) }
            await load()
            return true
        } catch {
            self.error = "Could not add the MCP server. Check the link and try again."
            return false
        }
    }
}

private enum CapabilitiesPage: String, CaseIterable, Identifiable {
    case skills = "Skills"
    case mcp = "MCP"

    var id: Self { self }
}

private struct CapabilitiesView: View {
    let port: Int
    @State private var selectedPage = CapabilitiesPage.skills

    var body: some View {
        VStack(spacing: 0) {
            HStack(alignment: .center, spacing: 18) {
                VStack(alignment: .leading, spacing: 3) {
                    Text("Capabilities")
                        .font(.system(size: 25, weight: .semibold, design: .rounded))
                    Text("Teach Jame new workflows and connect external tools.")
                        .font(.caption)
                        .foregroundStyle(JameBrand.muted)
                }
                Spacer()
                SquareSegmentedPicker(
                    options: CapabilitiesPage.allCases.map { (label: $0.rawValue, value: $0) },
                    selection: $selectedPage
                )
                .frame(width: 260)
            }
            .padding(.horizontal, 22)
            .padding(.vertical, 16)
            .background(JameBrand.panel)

            Rectangle().fill(JameBrand.rule).frame(height: 1)

            Group {
                switch selectedPage {
                case .skills:
                    SkillsView(port: port)
                case .mcp:
                    ConnectorsView(port: port)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
    }
}

private struct ConnectorsView: View {
    @StateObject private var store: ConnectorsStore
    @State private var showAddMCP = false

    init(port: Int) { _store = StateObject(wrappedValue: ConnectorsStore(port: port)) }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("MCP").font(.title2.weight(.semibold))
                        Text("MCP servers and CLI providers available to this agent.")
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    Button("Add MCP", systemImage: "plus") { showAddMCP = true }
                        .buttonStyle(.borderedProminent)
                    Button { Task { await store.load() } } label: { Image(systemName: "arrow.clockwise") }
                        .help("Refresh connectors")
                }

                ConnectorSection(title: "MCP servers", icon: "point.3.connected.trianglepath.dotted", empty: "No MCP servers are configured.") {
                    if store.mcpServers.isEmpty {
                        Text("No MCP servers are configured.").foregroundStyle(.secondary)
                    } else {
                        ForEach(store.mcpServers) { server in
                            ConnectorRow(name: server.name, detail: "\(server.transport.uppercased()) · \(server.endpoint)", connected: store.mcpEnabled && server.enabled)
                        }
                    }
                }

                ConnectorSection(title: "CLI providers", icon: "terminal", empty: "No CLI-backed models are configured.") {
                    if store.cliModels.isEmpty {
                        Text("No CLI-backed models are configured.").foregroundStyle(.secondary)
                    } else {
                        ForEach(store.cliModels) { model in
                            ConnectorRow(name: model.modelName, detail: [model.model, model.connectMode, model.workspace].compactMap { $0 }.joined(separator: " · "), connected: model.configured)
                        }
                    }
                }

                if store.isLoading { ProgressView("Loading MCP capabilities…") }
                if !store.error.isEmpty { Text(store.error).foregroundStyle(.red) }
            }
            .padding(24)
        }
        .task { await store.load() }
        .sheet(isPresented: $showAddMCP) {
            AddMCPServerSheet(store: store, isPresented: $showAddMCP)
        }
    }
}

private struct AddMCPServerSheet: View {
    @ObservedObject var store: ConnectorsStore
    @Binding var isPresented: Bool
    @State private var name = ""
    @State private var apiKey = ""
    @State private var url = ""
    @State private var isSaving = false

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            VStack(alignment: .leading, spacing: 5) {
                Text("Add MCP server").font(.title2.weight(.semibold))
                Text("Connect a remote MCP server so Jame can use its tools.")
                    .foregroundStyle(.secondary)
            }
            Form {
                TextField("Server name", text: $name, prompt: Text("e.g. linear"))
                SecureField("Server key (optional)", text: $apiKey, prompt: Text("API key or bearer token"))
                TextField("MCP link", text: $url, prompt: Text("https://example.com/mcp"))
            }
            Text("The server key is stored as an Authorization header and is not shown again in this screen.")
                .font(.caption)
                .foregroundStyle(.secondary)
            if !store.error.isEmpty { Text(store.error).font(.caption).foregroundStyle(.red) }
            HStack {
                Spacer()
                Button("Cancel") { isPresented = false }
                Button(isSaving ? "Adding…" : "Add MCP") {
                    isSaving = true
                    Task {
                        if await store.addMCPServer(name: name, apiKey: apiKey, url: url) { isPresented = false }
                        isSaving = false
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(isSaving)
            }
        }
        .padding(24)
        .frame(width: 500)
    }
}

private struct ConnectorSection<Content: View>: View {
    let title: String
    let icon: String
    let empty: String
    @ViewBuilder let content: Content

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Label(title, systemImage: icon).font(.headline)
            VStack(alignment: .leading, spacing: 8) { content }
                .padding(14)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(.quaternary, in: Rectangle())
        }
    }
}

private struct ConnectorRow: View {
    let name: String
    let detail: String
    let connected: Bool

    var body: some View {
        HStack(spacing: 10) {
            Circle().fill(connected ? .green : .gray).frame(width: 8, height: 8)
            VStack(alignment: .leading, spacing: 2) {
                Text(name).font(.subheadline.weight(.semibold))
                Text(detail.isEmpty ? "No connection details" : detail).font(.caption).foregroundStyle(.secondary).lineLimit(2)
            }
            Spacer()
            Text(connected ? "Connected" : "Not connected").font(.caption.weight(.medium)).foregroundStyle(connected ? .green : .secondary)
        }
    }
}

private func jameWorkspaceURL() -> URL {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let defaultWorkspace = home.appendingPathComponent(".jameclaw/workspace")
    let configURL = home.appendingPathComponent(".jameclaw/config.json")
    guard let data = try? Data(contentsOf: configURL),
          let config = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
          let agents = config["agents"] as? [String: Any],
          let defaults = agents["defaults"] as? [String: Any],
          let configuredPath = defaults["workspace"] as? String,
          !configuredPath.isEmpty else { return defaultWorkspace }
    if configuredPath.hasPrefix("~/") {
        return home.appendingPathComponent(String(configuredPath.dropFirst(2)))
    }
    return URL(fileURLWithPath: configuredPath)
}

private let jameTaskFolderDefaultsKey = "jameclaw.native-chat.task-folder"

private func jameInternalWorkspaceURL() -> URL {
    FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent(".jameclaw/workspace", isDirectory: true)
        .standardizedFileURL
}

private func jameTaskFolderURL() -> URL {
    if let storedPath = UserDefaults.standard.string(forKey: jameTaskFolderDefaultsKey), !storedPath.isEmpty {
        return URL(fileURLWithPath: storedPath, isDirectory: true).standardizedFileURL
    }
    let configured = jameWorkspaceURL().standardizedFileURL
    let internalWorkspace = jameInternalWorkspaceURL()
    if configured.path == internalWorkspace.path {
        return FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Documents/JameClaw", isDirectory: true)
            .standardizedFileURL
    }
    return configured
}

private func organizedJameTaskFolder(for selectedURL: URL) -> URL {
    let selected = selectedURL.standardizedFileURL.resolvingSymlinksInPath()
    let home = FileManager.default.homeDirectoryForCurrentUser
    let collectionFolders = ["Desktop", "Documents", "Downloads"].map {
        home.appendingPathComponent($0, isDirectory: true).standardizedFileURL.resolvingSymlinksInPath()
    }
    guard collectionFolders.contains(where: { $0.path == selected.path }) else { return selected }
    return selected.appendingPathComponent("JameClaw", isDirectory: true)
}

private struct WorkspaceEntry: Identifiable, Sendable {
    let url: URL
    let isDirectory: Bool
    var id: String { url.path }
}

@MainActor
private final class WorkspaceBrowser: ObservableObject {
    @Published var entries: [WorkspaceEntry] = []
    @Published var status = ""
    let title: String
    let directory: URL

    init(title: String, directory: URL) {
        self.title = title
        self.directory = directory
        refresh()
    }

    func refresh() {
        do {
            try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
            entries = try FileManager.default.contentsOfDirectory(
                at: directory,
                includingPropertiesForKeys: [.isDirectoryKey, .contentModificationDateKey],
                options: [.skipsHiddenFiles]
            )
            .map { url in
                let values = try? url.resourceValues(forKeys: [.isDirectoryKey])
                return WorkspaceEntry(url: url, isDirectory: values?.isDirectory ?? false)
            }
            .sorted { $0.url.lastPathComponent.localizedStandardCompare($1.url.lastPathComponent) == .orderedAscending }
            status = entries.isEmpty ? "Nothing here yet." : "\(entries.count) item\(entries.count == 1 ? "" : "s")"
        } catch {
            entries = []
            status = "Could not read this folder."
        }
    }

    func open() { NSWorkspace.shared.open(directory) }
}

private struct WorkspaceBrowserView: View {
    @ObservedObject var browser: WorkspaceBrowser
    let emptyTitle: String
    let emptyDescription: String

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                VStack(alignment: .leading, spacing: 3) {
                    Text(browser.title).font(.title3.weight(.semibold))
                    Text(browser.directory.path).font(.caption.monospaced()).foregroundStyle(.secondary).lineLimit(1)
                }
                Spacer()
                Button { browser.refresh() } label: { Image(systemName: "arrow.clockwise") }
                Button("Open Folder") { browser.open() }
            }
            .padding(18)
            Divider()
            if browser.entries.isEmpty {
                ContentUnavailableView(emptyTitle, systemImage: "tray", description: Text(emptyDescription))
            } else {
                List(browser.entries) { entry in
                    HStack(spacing: 10) {
                        Image(systemName: entry.isDirectory ? "folder.fill" : "doc.text")
                            .foregroundStyle(entry.isDirectory ? .orange : .secondary)
                        Text(entry.url.lastPathComponent).lineLimit(1)
                        Spacer()
                        Button("Show") { NSWorkspace.shared.activateFileViewerSelecting([entry.url]) }
                            .buttonStyle(.borderless)
                    }
                }
            }
            HStack { Text(browser.status).font(.caption).foregroundStyle(.secondary); Spacer() }
                .padding(.horizontal, 18).padding(.vertical, 10)
        }
        .task { browser.refresh() }
    }
}

private enum ArtifactsPage: String, CaseIterable, Identifiable {
    case projects
    case generatedImages

    var id: String { rawValue }

    var title: String {
        switch self {
        case .projects: return "Projects"
        case .generatedImages: return "Generated Images"
        }
    }
}

struct ArtifactsView: View {
    @StateObject private var browser = WorkspaceBrowser(
        title: "Artifacts",
        directory: jameWorkspaceURL().appendingPathComponent("artifacts")
    )
    @State private var selectedProject: WorkspaceEntry?
    @State private var selectedPage: ArtifactsPage = .projects

    private var projects: [WorkspaceEntry] {
        browser.entries.filter { $0.isDirectory && $0.url.lastPathComponent != "generated-images" }
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 12) {
                Text("Artifacts").font(.title3.weight(.semibold))
                Picker("Artifact view", selection: $selectedPage) {
                    ForEach(ArtifactsPage.allCases) { page in
                        Text(page.title).tag(page)
                    }
                }
                .pickerStyle(.segmented)
                .frame(maxWidth: 310)
                Spacer()
                Button { browser.refresh() } label: { Image(systemName: "arrow.clockwise") }
                Button("Open Folder") { browser.open() }
            }
            .padding(16)
            Divider()
            if selectedPage == .generatedImages {
                GeneratedImagesView(
                    directory: jameWorkspaceURL()
                        .appendingPathComponent("artifacts", isDirectory: true)
                        .appendingPathComponent("generated-images", isDirectory: true)
                )
            } else {
                HStack(spacing: 0) {
                    VStack(spacing: 0) {
                        if projects.isEmpty {
                            ContentUnavailableView(
                                "No artifacts yet",
                                systemImage: "shippingbox",
                                description: Text("Save a website artifact from the Web Console and it will appear here.")
                            )
                        } else {
                            List(projects) { project in
                                Button {
                                    selectedProject = project
                                } label: {
                                    Label(artifactTitle(project.url), systemImage: "folder.fill")
                                        .lineLimit(1)
                                        .foregroundStyle(selectedProject?.id == project.id ? .primary : .secondary)
                                }
                                .buttonStyle(.plain)
                                .padding(.vertical, 3)
                            }
                            .listStyle(.sidebar)
                        }
                    }
                    .frame(minWidth: 180, idealWidth: 240, maxWidth: 300)
                    Divider()
                    if let project = selectedProject {
                        ArtifactProjectView(projectURL: project.url, title: artifactTitle(project.url))
                            .id(project.id)
                    } else {
                        ContentUnavailableView(
                            "Choose an artifact",
                            systemImage: "folder",
                            description: Text("Select an artifact folder to view its files or start its website."))
                            .frame(maxWidth: .infinity, maxHeight: .infinity)
                    }
                }
            }
        }
        .task { browser.refresh() }
    }

    private func artifactTitle(_ directory: URL) -> String {
        let metadata = directory.appendingPathComponent("artifact.json")
        guard let data = try? Data(contentsOf: metadata),
              let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let name = object["name"] as? String,
              !name.isEmpty else { return directory.lastPathComponent }
        return name
    }
}

private struct GeneratedImageEntry: Identifiable {
    let url: URL
    let modifiedAt: Date

    var id: String { url.path }
}

private struct GeneratedImagesView: View {
    let directory: URL
    @State private var images: [GeneratedImageEntry] = []
    @State private var selectedImage: GeneratedImageEntry?
    @State private var status = ""

    private let columns = [
        GridItem(.adaptive(minimum: 180, maximum: 260), spacing: 14)
    ]

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                VStack(alignment: .leading, spacing: 3) {
                    Text("Images created by JameClaw").font(.headline)
                    Text("Every image generated by the agent is saved here.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button { refresh() } label: { Image(systemName: "arrow.clockwise") }
                Button("Open Folder") { NSWorkspace.shared.open(directory) }
            }
            .padding(18)
            Divider()
            if images.isEmpty {
                ContentUnavailableView(
                    "No generated images yet",
                    systemImage: "photo.on.rectangle.angled",
                    description: Text("Ask the agent to create an image and it will appear in this gallery."))
            } else {
                ScrollView {
                    LazyVGrid(columns: columns, spacing: 14) {
                        ForEach(images) { image in
                            Button { selectedImage = image } label: {
                                VStack(alignment: .leading, spacing: 8) {
                                    if let nsImage = NSImage(contentsOf: image.url) {
                                        Image(nsImage: nsImage)
                                            .resizable()
                                            .aspectRatio(contentMode: .fit)
                                            .frame(maxWidth: .infinity)
                                            .frame(height: 170)
                                            .background(.quaternary, in: Rectangle())
                                    } else {
                                        Image(systemName: "photo")
                                            .frame(maxWidth: .infinity)
                                            .frame(height: 170)
                                            .background(.quaternary, in: Rectangle())
                                    }
                                    Text(image.url.lastPathComponent)
                                        .font(.caption.weight(.medium))
                                        .lineLimit(1)
                                    Text(image.modifiedAt.formatted(date: .abbreviated, time: .shortened))
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                }
                                .padding(8)
                                .background(.background, in: Rectangle())
                                .overlay(Rectangle().stroke(.quaternary))
                                .contentShape(Rectangle())
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(18)
                }
            }
            HStack {
                Text(status).font(.caption).foregroundStyle(.secondary)
                Spacer()
            }
            .padding(.horizontal, 18)
            .padding(.vertical, 10)
        }
        .task { refresh() }
        .sheet(item: $selectedImage) { image in
            GeneratedImagePreview(image: image)
        }
    }

    private func refresh() {
        do {
            try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
            images = try FileManager.default.contentsOfDirectory(
                at: directory,
                includingPropertiesForKeys: [.contentModificationDateKey, .typeIdentifierKey],
                options: [.skipsHiddenFiles]
            )
            .filter { ["png", "jpg", "jpeg", "webp", "gif", "bmp", "tiff"].contains($0.pathExtension.lowercased()) }
            .compactMap { url in
                let values = try? url.resourceValues(forKeys: [.contentModificationDateKey])
                return GeneratedImageEntry(url: url, modifiedAt: values?.contentModificationDate ?? .distantPast)
            }
            .sorted { $0.modifiedAt > $1.modifiedAt }
            status = "\(images.count) image\(images.count == 1 ? "" : "s")"
        } catch {
            images = []
            status = "Could not read the generated images folder."
        }
    }
}

private struct GeneratedImagePreview: View {
    let image: GeneratedImageEntry

    var body: some View {
        VStack(spacing: 12) {
            if let nsImage = NSImage(contentsOf: image.url) {
                Image(nsImage: nsImage)
                    .resizable()
                    .aspectRatio(contentMode: .fit)
                    .frame(maxWidth: 900, maxHeight: 700)
            }
            Text(image.url.lastPathComponent).font(.caption).foregroundStyle(.secondary)
        }
        .padding(24)
        .frame(minWidth: 520, minHeight: 420)
    }
}

private struct ArtifactProjectView: View {
    let projectURL: URL
    let title: String
    @State private var files: [URL] = []
    @State private var selectedFile: URL?
    @State private var content = ""
    @State private var isRunning = false
    @State private var runID = UUID()
    @State private var status = ""

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 10) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(title).font(.title3.weight(.semibold))
                    Text(projectURL.path).font(.caption.monospaced()).foregroundStyle(.secondary).lineLimit(1)
                }
                Spacer()
                Button { loadFiles() } label: { Image(systemName: "arrow.clockwise") }
                if isRunning {
                    Button("Stop", systemImage: "stop.fill") { isRunning = false }
                } else {
                    Button("Start app", systemImage: "play.fill") {
                        guard FileManager.default.fileExists(atPath: indexURL.path) else {
                            status = "This artifact does not have an index.html file yet."
                            return
                        }
                        runID = UUID()
                        isRunning = true
                    }
                    .buttonStyle(.borderedProminent)
                }
            }
            .padding(18)
            Divider()
            if isRunning {
                ArtifactWebView(url: indexURL).id(runID)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                HStack(spacing: 0) {
                    List(files, id: \.path) { file in
                        Button {
                            select(file)
                        } label: {
                            Label(file.lastPathComponent, systemImage: file.pathExtension == "html" ? "chevron.left.forwardslash.chevron.right" : "doc.text")
                                .foregroundStyle(selectedFile?.path == file.path ? .primary : .secondary)
                        }
                        .buttonStyle(.plain)
                    }
                    .frame(minWidth: 150, idealWidth: 200, maxWidth: 250)
                    Divider()
                    VStack(alignment: .leading, spacing: 10) {
                        HStack {
                            Text(selectedFile?.lastPathComponent ?? "Select a file").font(.headline)
                            Spacer()
                            if selectedFile != nil {
                                Button("Save") { saveFile() }.buttonStyle(.borderedProminent)
                            }
                        }
                        if selectedFile != nil {
                            TextEditor(text: $content)
                                .font(.system(.body, design: .monospaced))
                                .scrollContentBackground(.hidden)
                                .padding(8)
                                .background(.quaternary, in: Rectangle())
                        } else {
                            ContentUnavailableView("Choose a project file", systemImage: "doc.text")
                        }
                        if !status.isEmpty { Text(status).font(.caption).foregroundStyle(.secondary) }
                    }
                    .padding(16)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                }
            }
        }
        .task { loadFiles() }
    }

    private var indexURL: URL { projectURL.appendingPathComponent("index.html") }

    private func loadFiles() {
        files = ((try? FileManager.default.contentsOfDirectory(at: projectURL, includingPropertiesForKeys: nil, options: [.skipsHiddenFiles])) ?? [])
            .filter { !$0.hasDirectoryPath && $0.lastPathComponent != "artifact.json" }
            .sorted { $0.lastPathComponent.localizedStandardCompare($1.lastPathComponent) == .orderedAscending }
        if let current = selectedFile, files.contains(where: { $0.path == current.path }) {
            select(current)
        } else if let first = files.first {
            select(first)
        }
    }

    private func select(_ file: URL) {
        selectedFile = file
        content = (try? String(contentsOf: file, encoding: .utf8)) ?? ""
        status = ""
    }

    private func saveFile() {
        guard let selectedFile else { return }
        do {
            try content.write(to: selectedFile, atomically: true, encoding: .utf8)
            status = "Saved \(selectedFile.lastPathComponent). Press Start app to run the latest files."
        } catch {
            status = "Could not save this file."
        }
    }
}

private struct ArtifactWebView: NSViewRepresentable {
    let url: URL

    func makeNSView(context: Context) -> WKWebView {
        let webView = WKWebView()
        webView.loadFileURL(url, allowingReadAccessTo: url.deletingLastPathComponent())
        return webView
    }

    func updateNSView(_ webView: WKWebView, context: Context) {}
}

private enum ProviderSetupPurpose {
    case primary
    case fallback
}

private struct ProviderSetupSheet: View {
    let port: Int
    @ObservedObject var providers: NativeProviderStore
    let purpose: ProviderSetupPurpose
    let done: () -> Void
    @State private var step: Step = .configure
    @State private var providerID = ""
    @State private var presetID = ""
    @State private var apiKey = ""

    private enum Step { case configure, select }

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text(step == .configure
                         ? (purpose == .primary ? "Add new AI provider" : "Add fallback provider")
                         : "Set global fallback provider")
                        .font(.title3.weight(.semibold))
                    Text(step == .configure
                         ? (purpose == .primary
                            ? "Connect a completely new provider and make it JameClaw's primary AI provider."
                            : "Configure a second provider, then choose it as JameClaw's fallback.")
                         : "This fallback is saved globally for Desktop, Web Console, agents, and channels.")
                        .font(.caption).foregroundStyle(.secondary)
                }
                Spacer()
                Button("Done", action: done)
                    .buttonStyle(.bordered)
            }
            .padding(16)
            Divider()
            if step == .configure {
                NativeProviderSetupForm(
                    providers: providers,
                    port: port,
                    providerID: $providerID,
                    presetID: $presetID,
                    apiKey: $apiKey,
                    setAsPrimary: purpose == .primary,
                    completed: {
                        if purpose == .primary { done() } else { step = .select }
                    }
                )
            } else {
                GlobalFallbackSelection(providers: providers, port: port, configureAgain: { step = .configure }, done: done)
            }
        }
        .frame(minWidth: 680, minHeight: 500)
    }
}

private struct NativeProviderSetupForm: View {
    @ObservedObject var providers: NativeProviderStore
    let port: Int
    @Binding var providerID: String
    @Binding var presetID: String
    @Binding var apiKey: String
    let setAsPrimary: Bool
    let completed: () -> Void
    @State private var discoveredModels: [NativeDiscoveredProviderModel] = []
    @State private var isDiscoveringModels = false

    private var selectedProvider: NativeProviderInfo? {
        providers.catalog.first(where: { $0.id == providerID })
    }
    private var presets: [NativeProviderModelPreset] { selectedProvider?.recommendedModels ?? [] }
    private var selectedPreset: NativeProviderModelPreset? { presets.first(where: { $0.id == presetID }) }
    private var selectedRemoteModel: NativeDiscoveredProviderModel? {
        guard presetID.hasPrefix("remote:") else { return nil }
        return discoveredModels.first(where: { $0.id == String(presetID.dropFirst("remote:".count)) })
    }
    private var requiresKey: Bool { selectedPreset?.requiresAPIKey ?? selectedProvider?.requiresAPIKey ?? false }

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Label("Configure a provider in Desktop", systemImage: "desktopcomputer.and.macbook")
                .font(.headline)
            Text("This setup writes directly to JameClaw's shared provider configuration. No Web Console is opened.")
                .font(.subheadline).foregroundStyle(.secondary)
            if providers.catalog.isEmpty {
                ProgressView("Loading available providers…")
                    .task { await providers.load(port: port) }
            } else {
                Form {
                    Picker("Provider", selection: $providerID) {
                        Text("Choose a provider").tag("")
                        ForEach(providers.catalog) { provider in
                            Text(provider.name).tag(provider.id)
                        }
                    }
                    .onChange(of: providerID) { _, _ in
                        presetID = presets.first?.id ?? ""
                        apiKey = ""
                        discoveredModels = []
                    }
                    Picker("Model", selection: $presetID) {
                        Text("Choose a model").tag("")
                        ForEach(presets) { preset in
                            Text(preset.name).tag(preset.id)
                        }
                        if !discoveredModels.isEmpty {
                            Section("Available from this API") {
                                ForEach(discoveredModels) { model in
                                    Text(model.name).tag("remote:\(model.id)")
                                }
                            }
                        }
                    }
                    if let provider = selectedProvider {
                        Text(provider.description).font(.caption).foregroundStyle(.secondary)
                    }
                    if requiresKey {
                        SecureField(selectedPreset?.keyLabel ?? selectedProvider?.keyLabel ?? "API key", text: $apiKey)
                    }
                }
                .formStyle(.grouped)
                HStack {
                    Button("Refresh providers") { Task { await providers.load(port: port) } }
                    Button(isDiscoveringModels ? "Fetching models…" : "Fetch available models") {
                        guard let provider = selectedProvider else { return }
                        Task {
                            isDiscoveringModels = true
                            discoveredModels = await providers.discoverCatalogModels(provider: provider, apiKey: apiKey, port: port)
                            isDiscoveringModels = false
                        }
                    }
                    .disabled(selectedProvider == nil || isDiscoveringModels)
                    Spacer()
                    Button(setAsPrimary ? "Add and use as primary" : "Add provider") {
                        guard let provider = selectedProvider, selectedPreset != nil || selectedRemoteModel != nil else { return }
                        Task {
                            if await providers.addCatalogModel(
                                provider: provider,
                                preset: selectedPreset,
                                remoteModelID: selectedRemoteModel?.id,
                                apiKey: apiKey,
                                setAsPrimary: setAsPrimary,
                                port: port
                            ) {
                                completed()
                            }
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(selectedProvider == nil || (selectedPreset == nil && selectedRemoteModel == nil) || (requiresKey && apiKey.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty))
                }
            }
            if !providers.status.isEmpty { Text(providers.status).font(.caption).foregroundStyle(.secondary) }
            Spacer()
        }
        .padding(24)
        .onAppear {
            if providerID.isEmpty { providerID = providers.catalog.first?.id ?? "" }
            if presetID.isEmpty { presetID = selectedProvider?.recommendedModels.first?.id ?? "" }
        }
        .onChange(of: providers.catalog.count) { _, _ in
            if providerID.isEmpty { providerID = providers.catalog.first?.id ?? "" }
            if presetID.isEmpty { presetID = selectedProvider?.recommendedModels.first?.id ?? "" }
        }
    }
}

private struct GlobalFallbackSelection: View {
    @ObservedObject var providers: NativeProviderStore
    let port: Int
    let configureAgain: () -> Void
    let done: () -> Void

    private var fallbackModels: [NativeModelInfo] {
        providers.models.filter { $0.modelName != providers.selectedModel }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Label("Global fallback coverage", systemImage: "point.3.connected.trianglepath.dotted")
                .font(.headline)
            Text("When the primary provider has a retriable error, every JameClaw entry point uses this fallback automatically: Desktop Chat, Web Console, team agents, automations, and connected channels.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
            if providers.models.count < 2 {
                ContentUnavailableView(
                    "Add one more provider first",
                    systemImage: "plus.circle",
                    description: Text("A global fallback must be different from the primary provider.")
                )
            } else {
                Form {
                    Picker("Primary provider", selection: $providers.selectedModel) {
                        ForEach(providers.models) { model in
                            Text("\(providers.providerName(for: model)) · \(model.modelName)").tag(model.modelName)
                        }
                    }
                    Picker("Global fallback", selection: $providers.selectedFallbackModel) {
                        Text("Choose a provider").tag("")
                        ForEach(fallbackModels) { model in
                            Text("\(providers.providerName(for: model)) · \(model.modelName)").tag(model.modelName)
                        }
                    }
                }
                .formStyle(.grouped)
                HStack {
                    Button("Back to provider setup", action: configureAgain)
                    Spacer()
                    Button("Apply global fallback") {
                        Task {
                            await providers.setFailover(
                                primaryModel: providers.selectedModel,
                                fallbackModel: providers.selectedFallbackModel,
                                port: port
                            )
                            if !providers.selectedFallbackModel.isEmpty { done() }
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(providers.selectedModel.isEmpty || providers.selectedFallbackModel.isEmpty)
                }
            }
            if !providers.status.isEmpty {
                Text(providers.status).font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
        }
        .padding(24)
    }
}

struct SkillsView: View {
    @StateObject private var browser = WorkspaceBrowser(
        title: "Skills",
        directory: jameWorkspaceURL().appendingPathComponent("skills")
    )
    @State private var showingAddSkill = false
    @State private var skillName = ""
    @State private var skillDescription = ""
    @State private var addError = ""
    @State private var healthItems: [NativeSkillHealth] = []
    @State private var curatorBusy = false
    private let port: Int

    init(port: Int) { self.port = port }

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                VStack(alignment: .leading, spacing: 3) {
                    Text("Skills").font(.title3.weight(.semibold))
                    Text(browser.directory.path).font(.caption.monospaced()).foregroundStyle(.secondary).lineLimit(1)
                }
                Spacer()
                Button { refreshAgentSkills() } label: { Image(systemName: "arrow.clockwise") }
                Button(curatorBusy ? "Curating…" : "Curate") {
                    Task { await curateSkills() }
                }
                .disabled(curatorBusy)
                Button("Add Skill") {
                    skillName = ""
                    skillDescription = ""
                    addError = ""
                    showingAddSkill = true
                }
                .buttonStyle(.borderedProminent)
                Button("Open Folder") { browser.open() }
            }
            .padding(18)
            Divider()
            if browser.entries.isEmpty && healthItems.isEmpty {
                ContentUnavailableView(
                    "No workspace skills yet",
                    systemImage: "wand.and.stars",
                    description: Text("Add a skill here, or use the Web Console to manage skills.")
                )
            } else {
                List {
                    if !healthItems.isEmpty {
                        Section("Skill health") {
                            ForEach(healthItems) { item in
                                HStack(spacing: 10) {
                                    Image(systemName: item.warnings.isEmpty ? "checkmark.seal.fill" : "exclamationmark.triangle.fill")
                                        .foregroundStyle(item.warnings.isEmpty ? .green : .orange)
                                    VStack(alignment: .leading, spacing: 3) {
                                        Text(item.name).font(.headline)
                                        Text(item.description.isEmpty ? "No description" : item.description)
                                            .font(.caption).foregroundStyle(.secondary).lineLimit(1)
                                    }
                                    Spacer()
                                    Text("\(item.usageCount) uses · \(item.successCount) ok · \(item.patchCount) changes")
                                        .font(.caption2.monospaced()).foregroundStyle(.secondary)
                                    Button("Show") { NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: item.path)]) }
                                        .buttonStyle(.borderless)
                                }
                                if !item.warnings.isEmpty {
                                    Text(item.warnings.map(\.message).joined(separator: " · "))
                                        .font(.caption2).foregroundStyle(.orange).lineLimit(2)
                                }
                            }
                        }
                    }
                    Section("Workspace files") {
                        ForEach(browser.entries) { entry in
                            HStack(spacing: 10) {
                                Image(systemName: entry.isDirectory ? "folder.fill" : "doc.text")
                                    .foregroundStyle(entry.isDirectory ? .orange : .secondary)
                                Text(entry.url.lastPathComponent).lineLimit(1)
                                Spacer()
                                Button("Show") { NSWorkspace.shared.activateFileViewerSelecting([entry.url]) }
                                    .buttonStyle(.borderless)
                            }
                        }
                    }
                }
            }
            HStack { Text(browser.status).font(.caption).foregroundStyle(.secondary); Spacer() }
                .padding(.horizontal, 18).padding(.vertical, 10)
        }
        .task { refreshAgentSkills() }
        .sheet(isPresented: $showingAddSkill) {
            VStack(alignment: .leading, spacing: 16) {
                Text("Add a Skill").font(.title2.weight(.semibold))
                Text("This creates a workspace skill that JameClaw loads from \(browser.directory.path).")
                    .foregroundStyle(.secondary)
                TextField("Skill name (for example, release-notes)", text: $skillName)
                TextField("What does this skill do?", text: $skillDescription, axis: .vertical)
                    .lineLimit(2...4)
                if !addError.isEmpty { Text(addError).font(.caption).foregroundStyle(.red) }
                HStack {
                    Spacer()
                    Button("Cancel") { showingAddSkill = false }
                    Button("Create Skill") { Task { await createSkill() } }
                        .buttonStyle(.borderedProminent)
                        .disabled(skillName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
            .padding(24)
            .frame(width: 480)
        }
    }

    private func createSkill() async {
        let displayName = skillName.trimmingCharacters(in: .whitespacesAndNewlines)
        let slug = displayName.lowercased()
            .replacingOccurrences(of: "[^a-z0-9]+", with: "-", options: .regularExpression)
            .trimmingCharacters(in: CharacterSet(charactersIn: "-"))
        guard !slug.isEmpty else {
            addError = "Use at least one letter or number in the skill name."
            return
        }
        let description = skillDescription.trimmingCharacters(in: .whitespacesAndNewlines)
        let document = """
        ---
        name: \(slug)
        description: \(description.isEmpty ? "Describe when JameClaw should use this skill." : description)
        ---

        # \(displayName)

        Add the instructions JameClaw should follow when using this skill.
        """
        do {
            var request = authenticatedConsoleRequest(port: port, path: "/api/skills/evolve", method: "POST")
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: ["action": "create", "name": slug, "content": document])
            let (data, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                let message = String(data: data, encoding: .utf8) ?? "The skill could not be created."
                throw NSError(domain: "JameClawSkills", code: 1, userInfo: [NSLocalizedDescriptionKey: message])
            }
            browser.refresh()
            await refreshAgentSkillsAsync()
            showingAddSkill = false
        } catch {
            addError = error.localizedDescription
        }
    }

    private func curateSkills() async {
        curatorBusy = true
        defer { curatorBusy = false }
        do {
            let request = authenticatedConsoleRequest(port: port, path: "/api/skills/curate", method: "POST")
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else { throw URLError(.badServerResponse) }
            await refreshAgentSkillsAsync()
        } catch {
            browser.status = "Could not curate skills: \(error.localizedDescription)"
        }
    }

    private func refreshAgentSkills() {
        Task {
            await refreshAgentSkillsAsync()
        }
    }

    private func refreshAgentSkillsAsync() async {
            do {
                let (data, response) = try await URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: "/api/skills"))
                guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else { throw URLError(.badServerResponse) }
                let skillList = try JSONDecoder().decode(NativeSkillsResponse.self, from: data).skills
                let directories = skillList.map { URL(fileURLWithPath: $0.path).deletingLastPathComponent() }
                browser.entries = directories.map { WorkspaceEntry(url: $0, isDirectory: true) }
                let healthData = try await URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: "/api/skills/health")).0
                healthItems = try JSONDecoder().decode([NativeSkillHealth].self, from: healthData)
                browser.status = "\(skillList.count) skills available · \(healthItems.filter { !$0.warnings.isEmpty }.count) need attention"
            } catch {
                browser.refresh()
                browser.status = "Showing workspace skills only. Could not reach the agent skill registry."
            }
    }
}

struct HomeView: View {
    @ObservedObject var settings: LauncherSettingsStore
    let openSettings: () -> Void
    @State private var status = "Starting your agent…"
    @State private var busy = false

    var body: some View {
        VStack(spacing: 18) {
            Image(systemName: "sparkles").font(.system(size: 34)).foregroundStyle(.red)
            Text("Jame").font(.title2.weight(.semibold))
            Text(status).multilineTextAlignment(.center).foregroundStyle(.secondary)
            HStack(spacing: 10) {
                Button("Open Web Console") { openConsole() }
                Button(busy ? "Starting…" : "Start agent") { startAgent() }
                    .keyboardShortcut(.defaultAction)
                    .disabled(busy)
            }
            Button("Settings") { openSettings() }
                .buttonStyle(.link)
        }
        .padding(30)
        .frame(width: 330)
        .task { startAgent() }
    }

    private func startAgent() {
        busy = true
        status = "Starting your agent…"
        Task {
            var request = URLRequest(url: authenticatedConsoleURL(port: Int(settings.port) ?? 18800, path: "/api/gateway/start"))
            request.httpMethod = "POST"
            do {
                let (_, response) = try await URLSession.shared.data(for: request)
                if let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) {
                    status = "Your agent is ready."
                } else {
                    status = "Jame is starting in the background."
                }
            } catch {
                status = "Jame is starting in the background."
            }
            busy = false
        }
    }

    private func openConsole() {
        NSWorkspace.shared.open(authenticatedConsoleURL(port: Int(settings.port) ?? 18800))
    }

}

private struct NativeAgentMemory: Codable {
    let longTerm: String
    let memoryPath: String

    enum CodingKeys: String, CodingKey {
        case longTerm = "long_term"
        case memoryPath = "memory_path"
    }
}

private struct NativeLearningCandidate: Codable, Identifiable {
    let id: String
    let kind: String
    let title: String
    let lesson: String
    let evidence: String
    let scope: String
    let confidence: Double
    let status: String
    let occurrences: Int
    let requiresApproval: Bool
    let tools: [String]?
    let skillPath: String?
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id, kind, title, lesson, evidence, scope, confidence, status, occurrences, tools
        case requiresApproval = "requires_approval"
        case skillPath = "skill_path"
        case updatedAt = "updated_at"
    }
}

private struct NativeTurnReflection: Codable, Identifiable {
    let id: String
    let objective: String
    let outcome: String
    let resultSummary: String
    let tools: [String]?
    let toolFailures: [String]?
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case id, objective, outcome, tools
        case resultSummary = "result_summary"
        case toolFailures = "tool_failures"
        case createdAt = "created_at"
    }
}

private struct NativeImprovementMetrics: Codable {
    let reflections: Int
    let pendingCandidates: Int
    let promotedCandidates: Int
    let repeatedFailureCount: Int
    let skillsCreated: Int
    let completionRate: Double
    let correctionRate: Double

    enum CodingKeys: String, CodingKey {
        case reflections
        case pendingCandidates = "pending_candidates"
        case promotedCandidates = "promoted_candidates"
        case repeatedFailureCount = "repeated_failure_count"
        case skillsCreated = "skills_created"
        case completionRate = "completion_rate"
        case correctionRate = "correction_rate"
    }

    static let empty = NativeImprovementMetrics(
        reflections: 0, pendingCandidates: 0, promotedCandidates: 0,
        repeatedFailureCount: 0, skillsCreated: 0, completionRate: 0, correctionRate: 0
    )
}

private struct NativeSelfImprovementSnapshot: Codable {
    let candidates: [NativeLearningCandidate]
    let reflections: [NativeTurnReflection]
    let metrics: NativeImprovementMetrics
}

@MainActor
private final class NativeMemoryStore: ObservableObject {
    @Published var text = ""
    @Published var path = ""
    @Published var status = ""
    @Published var agentName = ""
    @Published var persona = ""
    @Published var tone = ""
    @Published var discussionMode = ""
    @Published var configuredMemoryNotes = ""
    @Published var statusStyle = ""
    @Published var userProfile = ""
    @Published var peopleAndRelationships = ""
    @Published var improvementCandidates: [NativeLearningCandidate] = []
    @Published var reflections: [NativeTurnReflection] = []
    @Published var improvementMetrics = NativeImprovementMetrics.empty
    private let port: Int
    init(port: Int) { self.port = port }

    func load() async {
        do {
            let (data, response) = try await URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: "/api/agents/main/memory"))
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else { throw URLError(.badServerResponse) }
            let memory = try JSONDecoder().decode(NativeAgentMemory.self, from: data)
            text = memory.longTerm
            userProfile = memorySection("User Profile", in: text)
            peopleAndRelationships = memorySection("People & Relationships", in: text)
            path = memory.memoryPath
            let (agentData, _) = try await URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: "/api/agents"))
            if let main = try? JSONDecoder().decode(NativeAgentsResponse.self, from: agentData).agents.first(where: { $0.id == "main" }) {
                agentName = main.name
                persona = main.human?.persona ?? ""
                tone = main.human?.tone ?? ""
                discussionMode = main.human?.discussionMode ?? ""
                configuredMemoryNotes = main.human?.memoryNotes ?? ""
                statusStyle = main.human?.statusStyle ?? ""
            }
            await loadSelfImprovement()
            status = ""
        } catch { status = "Could not load agent memory." }
    }

    func loadSelfImprovement() async {
        do {
            let (data, response) = try await URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: "/api/agents/main/self-improvement"))
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else { throw URLError(.badServerResponse) }
            let snapshot = try JSONDecoder().decode(NativeSelfImprovementSnapshot.self, from: data)
            improvementCandidates = snapshot.candidates
            reflections = snapshot.reflections
            improvementMetrics = snapshot.metrics
        } catch {
            status = "Could not load self-improvement history."
        }
    }

    func updateCandidate(_ candidate: NativeLearningCandidate, action: String) async {
        do {
            let candidateID = candidate.id.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? candidate.id
            var request = authenticatedConsoleRequest(port: port, path: "/api/agents/main/self-improvement/candidates/\(candidateID)", method: "PUT")
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: ["action": action])
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else { throw URLError(.badServerResponse) }
            await loadSelfImprovement()
            status = action == "create_skill" ? "Reusable skill created." : "Learning decision saved."
        } catch {
            status = "Could not update this learning candidate."
        }
    }

    func runMaintenance() async {
        do {
            let request = authenticatedConsoleRequest(port: port, path: "/api/agents/main/self-improvement/maintenance", method: "POST")
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else { throw URLError(.badServerResponse) }
            await loadSelfImprovement()
            status = "Learning history reviewed."
        } catch {
            status = "Could not run learning maintenance."
        }
    }

    func saveIdentity() async {
        do {
            var request = authenticatedConsoleRequest(port: port, path: "/api/agents/main", method: "PATCH")
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONEncoder().encode(NativeUpdateAgentRequest(human: NativeUpdateAgentHuman(agentName: agentName, persona: persona, tone: tone, discussionMode: discussionMode, memoryNotes: configuredMemoryNotes, statusStyle: statusStyle)))
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else { throw URLError(.badServerResponse) }
            status = "Identity saved."
        } catch { status = "Could not save identity." }
    }

    func save() async {
        do {
            var request = authenticatedConsoleRequest(port: port, path: "/api/agents/main/memory", method: "PUT")
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: ["long_term": text])
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else { throw URLError(.badServerResponse) }
            status = "Memory saved."
        } catch { status = "Could not save agent memory." }
    }

    func saveUserContext() async {
        text = replacingMemorySection("User Profile", with: userProfile, in: text)
        text = replacingMemorySection("People & Relationships", with: peopleAndRelationships, in: text)
        await save()
    }
}

private func memorySection(_ title: String, in text: String) -> String {
    let escapedTitle = NSRegularExpression.escapedPattern(for: title)
    let pattern = "(?ms)^#{1,6}\\s+\(escapedTitle)\\s*$\\n?(.*?)(?=^#|\\z)"
    guard let range = text.range(of: pattern, options: .regularExpression) else { return "" }
    let block = String(text[range])
    return block
        .replacingOccurrences(of: "(?m)^#{1,6}\\s+\(escapedTitle)\\s*$", with: "", options: .regularExpression)
        .trimmingCharacters(in: .whitespacesAndNewlines)
}

private func replacingMemorySection(_ title: String, with value: String, in text: String) -> String {
    let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
    let block = trimmed.isEmpty ? "" : "# \(title)\n\(trimmed)\n"
    let escapedTitle = NSRegularExpression.escapedPattern(for: title)
    let pattern = "(?ms)^#{1,6}\\s+\(escapedTitle)\\s*$\\n?.*?(?=^#|\\z)"
    let range = text.range(of: pattern, options: .regularExpression)
    let withoutOld = range.map { text.replacingCharacters(in: $0, with: "") } ?? text
    return (withoutOld.trimmingCharacters(in: .whitespacesAndNewlines) + "\n\n" + block)
        .trimmingCharacters(in: .whitespacesAndNewlines) + "\n"
}

private struct AgentMemoryView: View {
    @StateObject private var store: NativeMemoryStore
    @State private var selectedTab = "memory"
    @Environment(\.colorScheme) private var colorScheme

    private var pageBackground: Color { colorScheme == .light ? Color(red: 0.965, green: 0.965, blue: 0.955) : JameBrand.ink }
    private var panel: Color { colorScheme == .light ? .white : JameBrand.panel }
    private var primary: Color { colorScheme == .light ? JameBrand.ink : JameBrand.paper }
    private var muted: Color { colorScheme == .light ? JameBrand.ink.opacity(0.58) : JameBrand.muted }
    private var rule: Color { colorScheme == .light ? JameBrand.ink.opacity(0.14) : JameBrand.rule }

    init(port: Int) { _store = StateObject(wrappedValue: NativeMemoryStore(port: port)) }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 5) {
                    Text("LOCAL CONTEXT")
                        .font(.system(size: 10, weight: .semibold, design: .rounded))
                        .tracking(1.4)
                        .foregroundStyle(JameBrand.orange)
                    Text("Memory")
                        .font(.system(size: 28, weight: .semibold, design: .rounded))
                    Text("The durable facts Jame can carry into a new conversation.")
                        .font(.subheadline)
                        .foregroundStyle(muted)
                }
                Spacer()
                Button { Task { await store.load() } } label: { Label("Reload", systemImage: "arrow.clockwise") }
                    .buttonStyle(.bordered)
            }
            .padding(.horizontal, 24).padding(.top, 22).padding(.bottom, 18)

            HStack(spacing: 24) {
                VStack(alignment: .leading, spacing: 3) {
                    Text("\(store.text.count.formatted())")
                        .font(.system(size: 20, weight: .semibold, design: .rounded))
                        .foregroundStyle(primary)
                    Text("CHARACTERS SAVED").font(.system(size: 9, weight: .semibold, design: .rounded)).foregroundStyle(muted)
                }
                Rectangle().fill(rule).frame(width: 1, height: 34)
                VStack(alignment: .leading, spacing: 3) {
                    Text("Relevant snippets")
                        .font(.system(size: 14, weight: .semibold, design: .rounded))
                        .foregroundStyle(JameBrand.orange)
                    Text("RECALLED PER MESSAGE").font(.system(size: 9, weight: .semibold, design: .rounded)).foregroundStyle(muted)
                }
                Spacer()
                Text(store.path.isEmpty ? "memory/MEMORY.md" : store.path)
                    .font(.caption2.monospaced())
                    .foregroundStyle(muted)
                    .lineLimit(1).truncationMode(.middle)
                    .frame(maxWidth: 360)
            }
            .padding(.horizontal, 24).padding(.vertical, 13)
            .background(panel)
            .overlay(alignment: .leading) { Rectangle().fill(JameBrand.orange).frame(width: 3) }

            SquareSegmentedPicker(
                options: [
                    (label: "Long-term memory", value: "memory"),
                    (label: "Identity & preferences", value: "identity"),
                    (label: "Self-improvement", value: "improvement"),
                ],
                selection: $selectedTab
            )
            .frame(maxWidth: 590)
            .padding(.horizontal, 24).padding(.vertical, 16)

            Rectangle().fill(rule).frame(height: 1)

            if selectedTab == "memory" {
                VStack(alignment: .leading, spacing: 12) {
                    HStack {
                        VStack(alignment: .leading, spacing: 3) {
                            Text("Durable memory").font(.headline)
                            Text("Keep stable preferences, decisions, people, and recurring workflows here. Daily task logs should stay out.")
                                .font(.caption).foregroundStyle(muted)
                        }
                        Spacer()
                        Button("Save memory") { Task { await store.save() } }
                            .buttonStyle(.borderedProminent)
                            .tint(JameBrand.orange)
                    }
                    TextEditor(text: $store.text)
                        .font(.system(size: 13, design: .monospaced))
                        .scrollContentBackground(.hidden)
                        .padding(8)
                        .background(panel, in: Rectangle())
                        .overlay(Rectangle().stroke(rule))
                }
                .padding(24)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if selectedTab == "identity" {
                ScrollView {
                    VStack(alignment: .leading, spacing: 18) {
                        memoryFieldGroup("Agent identity", detail: "How Jame presents itself in every discussion") {
                            TextField("Agent name", text: $store.agentName)
                            TextField("Persona", text: $store.persona, axis: .vertical).lineLimit(2...4)
                            TextField("Tone", text: $store.tone)
                            TextField("Discussion mode", text: $store.discussionMode)
                            TextField("Status style", text: $store.statusStyle)
                        }
                        memoryFieldGroup("Configured notes", detail: "Short standing instructions stored with the agent configuration") {
                            memoryEditor($store.configuredMemoryNotes, minHeight: 90)
                        }
                        memoryFieldGroup("User profile", detail: "Preferences, goals, working style, and durable context") {
                            memoryEditor($store.userProfile, minHeight: 120)
                        }
                        memoryFieldGroup("People & relationships", detail: "Only people the user explicitly mentioned and their role") {
                            memoryEditor($store.peopleAndRelationships, minHeight: 110)
                        }
                        HStack {
                            Spacer()
                            Button("Save agent identity") { Task { await store.saveIdentity() } }
                            Button("Save user context") { Task { await store.saveUserContext() } }
                                .buttonStyle(.borderedProminent).tint(JameBrand.orange)
                        }
                    }
                    .padding(24)
                }
            } else {
                selfImprovementView
            }

            if !store.status.isEmpty {
                Text(store.status)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(store.status.contains("saved") ? JameBrand.orange : .red)
                    .padding(.horizontal, 24).padding(.bottom, 12)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .foregroundStyle(primary)
        .background(pageBackground)
        .task { await store.load() }
    }

    private var selfImprovementView: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                HStack(spacing: 0) {
                    improvementMetric("COMPLETION", value: "\(Int((store.improvementMetrics.completionRate * 100).rounded()))%")
                    Rectangle().fill(rule).frame(width: 1, height: 44)
                    improvementMetric("PENDING REVIEW", value: "\(store.improvementMetrics.pendingCandidates)")
                    Rectangle().fill(rule).frame(width: 1, height: 44)
                    improvementMetric("LEARNED", value: "\(store.improvementMetrics.promotedCandidates)")
                    Rectangle().fill(rule).frame(width: 1, height: 44)
                    improvementMetric("SKILLS CREATED", value: "\(store.improvementMetrics.skillsCreated)")
                    Rectangle().fill(rule).frame(width: 1, height: 44)
                    improvementMetric("REPEATED ERRORS", value: "\(store.improvementMetrics.repeatedFailureCount)")
                }
                .background(panel)
                .overlay(Rectangle().stroke(rule))

                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Learning review").font(.headline)
                        Text("Jame reflects after completed tasks. Explicit memory requests are learned immediately; corrections and reusable workflows stay here until you approve them.")
                            .font(.caption).foregroundStyle(muted)
                    }
                    Spacer()
                    Button("Run maintenance") { Task { await store.runMaintenance() } }
                        .buttonStyle(.bordered)
                }

                if store.improvementCandidates.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("No learning candidates yet").font(.headline)
                        Text("Candidates will appear after corrections, repeated tool failures, preferences, and repeated successful workflows.")
                            .font(.caption).foregroundStyle(muted)
                    }
                    .padding(16).frame(maxWidth: .infinity, alignment: .leading)
                    .background(panel).overlay(Rectangle().stroke(rule))
                } else {
                    ForEach(store.improvementCandidates) { candidate in
                        improvementCandidateCard(candidate)
                    }
                }

                if !store.reflections.isEmpty {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Recent reflections").font(.headline)
                        ForEach(Array(store.reflections.prefix(8))) { reflection in
                            VStack(alignment: .leading, spacing: 5) {
                                HStack {
                                    Text(reflection.outcome.uppercased())
                                        .font(.system(size: 9, weight: .semibold, design: .rounded))
                                        .foregroundStyle(reflection.outcome == "completed" ? JameBrand.orange : muted)
                                    Spacer()
                                    Text(reflection.createdAt).font(.caption2.monospaced()).foregroundStyle(muted)
                                }
                                Text(reflection.objective).font(.subheadline.weight(.semibold)).lineLimit(2)
                                if let tools = reflection.tools, !tools.isEmpty {
                                    Text("Tools: " + tools.joined(separator: " · ")).font(.caption).foregroundStyle(muted)
                                }
                            }
                            .padding(12).background(panel).overlay(Rectangle().stroke(rule))
                        }
                    }
                }
            }
            .padding(24)
        }
    }

    private func improvementMetric(_ title: String, value: String) -> some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(value).font(.system(size: 19, weight: .semibold, design: .rounded)).foregroundStyle(primary)
            Text(title).font(.system(size: 8, weight: .semibold, design: .rounded)).foregroundStyle(muted)
        }
        .padding(.horizontal, 14).padding(.vertical, 11).frame(maxWidth: .infinity, alignment: .leading)
    }

    private func improvementCandidateCard(_ candidate: NativeLearningCandidate) -> some View {
        VStack(alignment: .leading, spacing: 9) {
            HStack {
                Text(candidate.kind.replacingOccurrences(of: "_", with: " ").uppercased())
                    .font(.system(size: 9, weight: .semibold, design: .rounded)).tracking(0.8)
                    .foregroundStyle(JameBrand.orange)
                Text(candidate.status.uppercased())
                    .font(.system(size: 9, weight: .semibold, design: .rounded)).foregroundStyle(muted)
                Spacer()
                Text("\(Int((candidate.confidence * 100).rounded()))% confidence · \(candidate.occurrences)x")
                    .font(.caption2.monospaced()).foregroundStyle(muted)
            }
            Text(candidate.title).font(.headline)
            Text(candidate.lesson).font(.subheadline).textSelection(.enabled)
            Text(candidate.evidence).font(.caption).foregroundStyle(muted).lineLimit(3)
            if let tools = candidate.tools, !tools.isEmpty {
                Text("Observed tools: " + tools.joined(separator: " · ")).font(.caption).foregroundStyle(muted)
            }
            HStack {
                Text("Scope: \(candidate.scope)").font(.caption2).foregroundStyle(muted)
                Spacer()
                if candidate.status == "pending" || candidate.status == "stale" {
                    Button("Reject") { Task { await store.updateCandidate(candidate, action: "reject") } }
                        .buttonStyle(.bordered)
                    Button("Approve") { Task { await store.updateCandidate(candidate, action: "approve") } }
                        .buttonStyle(.borderedProminent).tint(JameBrand.orange)
                } else if candidate.status == "promoted", (candidate.skillPath ?? "").isEmpty {
                    Button("Create reusable skill") { Task { await store.updateCandidate(candidate, action: "create_skill") } }
                        .buttonStyle(.bordered).tint(JameBrand.orange)
                } else if let skillPath = candidate.skillPath, !skillPath.isEmpty {
                    Text(skillPath).font(.caption2.monospaced()).foregroundStyle(JameBrand.orange).lineLimit(1)
                }
            }
        }
        .padding(14)
        .background(panel)
        .overlay(alignment: .leading) { Rectangle().fill(candidate.status == "promoted" ? JameBrand.orange : rule).frame(width: 3) }
        .overlay(Rectangle().stroke(rule))
    }

    private func memoryEditor(_ value: Binding<String>, minHeight: CGFloat) -> some View {
        TextEditor(text: value)
            .font(.system(.body, design: .monospaced))
            .scrollContentBackground(.hidden)
            .frame(minHeight: minHeight)
            .padding(7)
            .background(panel, in: Rectangle())
            .overlay(Rectangle().stroke(rule))
    }

    private func memoryFieldGroup<Content: View>(
        _ title: String,
        detail: String,
        @ViewBuilder content: () -> Content
    ) -> some View {
        VStack(alignment: .leading, spacing: 9) {
            Text(title).font(.headline)
            Text(detail).font(.caption).foregroundStyle(muted)
            content()
        }
        .padding(.leading, 14)
        .overlay(alignment: .leading) { Rectangle().fill(JameBrand.orange.opacity(0.72)).frame(width: 2) }
    }
}

private struct NativeAgentHuman: Codable {
    let agentName: String?
    let persona: String?
    let tone: String?
    let discussionMode: String?
    let memoryNotes: String?
    let statusStyle: String?

    enum CodingKeys: String, CodingKey {
        case agentName = "agent_name"
        case persona, tone
        case discussionMode = "discussion_mode"
        case memoryNotes = "memory_notes"
        case statusStyle = "status_style"
    }
}

private struct NativeAgentSummary: Codable, Identifiable {
    let id: String
    let name: String
    let isDefault: Bool
    let workspace: String
    let model: String
    let skills: [String]?
    let subagents: [String]?
    let human: NativeAgentHuman?
    let sessionCount: Int?
    let messageCount: Int?
    let toolCalls: Int?

    enum CodingKeys: String, CodingKey {
        case id, name, workspace, model, skills, subagents, human
        case isDefault = "default"
        case sessionCount = "session_count"
        case messageCount = "message_count"
        case toolCalls = "tool_calls"
    }
}

private struct NativeAgentsResponse: Codable {
    let agents: [NativeAgentSummary]
}

private struct NativeCreateAgentRequest: Encodable {
    let id: String
    let name: String
    let workspace: String
    let managedByMain: Bool
    let parentID: String?
    let human: NativeCreateAgentHuman

    enum CodingKeys: String, CodingKey {
        case id, name, workspace, human
        case managedByMain = "managed_by_main"
        case parentID = "parent_id"
    }
}

private struct NativeCreateAgentHuman: Encodable {
    let agentName: String
    let persona: String
    let memoryNotes: String

    enum CodingKeys: String, CodingKey {
        case agentName = "agent_name"
        case persona
        case memoryNotes = "memory_notes"
    }
}

private struct NativeLocalAgent: Identifiable {
    let preset: TeamAgentPreset
    let location: String
    let runtimeProvider: String?
    let runtimeModel: String?

    var id: String { preset.id }
    var title: String { preset.title }
}

private struct NativeLocalAgentRuntime {
    let provider: String
    let model: String
}

private func agentIdentitySymbol(_ agent: NativeAgentSummary) -> String {
    if agent.id == "main" { return "sparkles" }
    if let preset = TeamAgentPreset(rawValue: agent.id) { return preset.symbol }
    let symbols = [
        "scope", "lightbulb.fill", "hammer.fill", "magnifyingglass",
        "chart.bar.fill", "shield.fill", "paperplane.fill", "terminal.fill",
        "book.closed.fill", "puzzlepiece.extension.fill", "wand.and.stars", "gearshape.2.fill",
    ]
    let stableIndex = agent.id.utf8.reduce(0) { (result, byte) in
        (result &* 31 &+ Int(byte)) % symbols.count
    }
    return symbols[stableIndex]
}

private func unquotedYAMLValue(_ rawValue: Substring) -> String {
    var value = rawValue.trimmingCharacters(in: .whitespacesAndNewlines)
    if let commentIndex = value.firstIndex(of: "#") {
        value = String(value[..<commentIndex]).trimmingCharacters(in: .whitespacesAndNewlines)
    }
    if value.count >= 2,
       let first = value.first,
       let last = value.last,
       (first == "\"" && last == "\"") || (first == "'" && last == "'") {
        value.removeFirst()
        value.removeLast()
    }
    return value
}

private func hermesRuntimeMetadata() -> NativeLocalAgentRuntime {
    // Hermes is a separate local runtime. Never infer its provider from
    // JameClaw's default model; read Hermes' own non-secret model metadata.
    let fallback = NativeLocalAgentRuntime(provider: "Nous Research", model: "")
    let configURL = FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent(".hermes/config.yaml")
    guard let contents = try? String(contentsOf: configURL, encoding: .utf8) else {
        return fallback
    }

    var insideModelSection = false
    var provider = ""
    var model = ""
    for rawLine in contents.split(whereSeparator: \.isNewline) {
        let line = String(rawLine)
        let trimmed = line.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, !trimmed.hasPrefix("#") else { continue }
        let isTopLevel = line.first.map { !$0.isWhitespace } ?? true
        if isTopLevel {
            insideModelSection = trimmed == "model:"
            continue
        }
        guard insideModelSection,
              let separator = trimmed.firstIndex(of: ":") else { continue }
        let key = trimmed[..<separator].trimmingCharacters(in: .whitespacesAndNewlines)
        let value = unquotedYAMLValue(trimmed[trimmed.index(after: separator)...])
        switch key {
        case "provider": provider = value
        case "default": model = value
        default: break
        }
    }

    let providerName: String
    switch provider.lowercased() {
    case "nous": providerName = "Nous Research"
    case "openrouter": providerName = "OpenRouter"
    case "openai": providerName = "OpenAI"
    case "anthropic": providerName = "Anthropic"
    default: providerName = provider.isEmpty ? fallback.provider : provider
    }
    return NativeLocalAgentRuntime(provider: providerName, model: model)
}

private func detectedLocalAgents() -> [NativeLocalAgent] {
    let fileManager = FileManager.default
    let pathDirectories = (ProcessInfo.processInfo.environment["PATH"] ?? "")
        .split(separator: ":")
        .map { URL(fileURLWithPath: String($0), isDirectory: true) }
    let commonExecutableDirectories = [
        URL(fileURLWithPath: "/opt/homebrew/bin", isDirectory: true),
        URL(fileURLWithPath: "/usr/local/bin", isDirectory: true),
        FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".local/bin", isDirectory: true),
        FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".npm-global/bin", isDirectory: true),
        FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".bun/bin", isDirectory: true),
        FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".cargo/bin", isDirectory: true),
        FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("bin", isDirectory: true),
    ]
    var seenExecutableDirectories = Set<String>()
    let executableDirectories = (pathDirectories + commonExecutableDirectories).filter {
        seenExecutableDirectories.insert($0.path).inserted
    }
    let applicationDirectories = [
        URL(fileURLWithPath: "/Applications", isDirectory: true),
        URL(fileURLWithPath: "/System/Applications", isDirectory: true),
        FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("Applications", isDirectory: true),
    ]
    let candidates: [(TeamAgentPreset, [String], [String])] = [
        (.codex, ["codex"], ["Codex.app", "ChatGPT.app"]),
        (.kimi, ["kimi", "kimi-code"], ["Kimi.app"]),
        (.claudeCode, ["claude"], ["Claude.app"]),
        (.hermes, ["hermes"], ["Hermes.app"]),
        (.grok, ["grok", "xai"], []),
        (.nanobot, ["nanobot"], []),
        (.openCode, ["opencode"], ["OpenCode.app"]),
        (.aider, ["aider"], []),
        (.gemini, ["gemini"], ["Gemini.app"]),
        (.goose, ["goose"], ["Goose.app"]),
        (.qwenCode, ["qwen", "qwen-code"], []),
        (.cursor, ["cursor-agent", "cursor"], ["Cursor.app"]),
    ]

    return candidates.compactMap { preset, commands, applications in
        let runtime = preset == .hermes ? hermesRuntimeMetadata() : nil
        if let commandPath = commands.lazy.compactMap({ command in
            executableDirectories
                .map { $0.appendingPathComponent(command).path }
                .first(where: { fileManager.isExecutableFile(atPath: $0) })
        }).first {
            return NativeLocalAgent(
                preset: preset,
                location: commandPath,
                runtimeProvider: runtime?.provider,
                runtimeModel: runtime?.model
            )
        }
        if let appPath = discoveredMacApp(named: applications, in: applicationDirectories, fileManager: fileManager) {
            return NativeLocalAgent(
                preset: preset,
                location: appPath,
                runtimeProvider: runtime?.provider,
                runtimeModel: runtime?.model
            )
        }
        return nil
    }
}

private func discoveredMacApp(named appNames: [String], in directories: [URL], fileManager: FileManager) -> String? {
    guard !appNames.isEmpty else { return nil }
    let wanted = Set(appNames.map { $0.lowercased() })
    for directory in directories {
        // Direct lookup is fast for normal /Applications installs.
        for name in appNames {
            let path = directory.appendingPathComponent(name).path
            if fileManager.fileExists(atPath: path) { return path }
        }
        // Some app managers place apps one folder deeper. Search only package
        // names, skip descendants, and stop at the first matching app bundle.
        guard let enumerator = fileManager.enumerator(at: directory, includingPropertiesForKeys: [.isDirectoryKey], options: [.skipsHiddenFiles, .skipsPackageDescendants]) else { continue }
        for case let url as URL in enumerator where url.pathExtension.lowercased() == "app" {
            if wanted.contains(url.lastPathComponent.lowercased()) {
                return url.path
            }
            enumerator.skipDescendants()
        }
    }
    return nil
}

private struct NativeUpdateAgentRequest: Encodable {
    let human: NativeUpdateAgentHuman
}

private struct NativeUpdateAgentHuman: Encodable {
    let agentName: String
    let persona: String
    let tone: String
    let discussionMode: String
    let memoryNotes: String
    let statusStyle: String

    enum CodingKeys: String, CodingKey {
        case agentName = "agent_name"
        case persona, tone
        case discussionMode = "discussion_mode"
        case memoryNotes = "memory_notes"
        case statusStyle = "status_style"
    }
}

@MainActor
private final class NativeAgentStore: ObservableObject {
    @Published var agents: [NativeAgentSummary] = []
    @Published var localAgents: [NativeLocalAgent] = []
    @Published var isLoading = false
    @Published var error = ""
    @Published var isCreating = false

    private let port: Int

    init(port: Int) { self.port = port }

    func discoverLocalAgents() {
        localAgents = detectedLocalAgents()
    }

    func load() async {
        discoverLocalAgents()
        isLoading = true
        error = ""
        defer { isLoading = false }
        do {
            let (data, response) = try await URLSession.shared.data(
                from: authenticatedConsoleURL(port: port, path: "/api/agents")
            )
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            agents = try JSONDecoder().decode(NativeAgentsResponse.self, from: data).agents
        } catch {
            self.error = "Could not load agents. Start JameClaw and try again."
        }
    }

    func createAgent(id: String, name: String, workspace: String, persona: String, memoryNotes: String, parentID: String?, managedByMain: Bool) async -> Bool {
        let cleanID = id.trimmingCharacters(in: .whitespacesAndNewlines)
        let cleanName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !cleanID.isEmpty else {
            error = "Enter an agent ID."
            return false
        }
        isCreating = true
        error = ""
        defer { isCreating = false }
        do {
            var request = authenticatedConsoleRequest(port: port, path: "/api/agents", method: "POST")
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONEncoder().encode(
                NativeCreateAgentRequest(
                    id: cleanID,
                    name: cleanName.isEmpty ? cleanID : cleanName,
                    workspace: workspace.trimmingCharacters(in: .whitespacesAndNewlines),
                    managedByMain: managedByMain,
                    parentID: parentID,
                    human: NativeCreateAgentHuman(
                        agentName: cleanName.isEmpty ? cleanID : cleanName,
                        persona: persona.trimmingCharacters(in: .whitespacesAndNewlines),
                        memoryNotes: memoryNotes.trimmingCharacters(in: .whitespacesAndNewlines)
                    )
                )
            )
            let (data, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                let message = String(data: data, encoding: .utf8) ?? "Could not create agent."
                throw NSError(domain: "JameClaw", code: 1, userInfo: [NSLocalizedDescriptionKey: message])
            }
            await load()
            return true
        } catch {
            self.error = error.localizedDescription.isEmpty ? "Could not create agent." : error.localizedDescription
            return false
        }
    }

    func rename(_ agent: NativeAgentSummary, to name: String) async -> Bool {
        let cleanName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !cleanName.isEmpty else {
            error = "Enter an agent name."
            return false
        }
        error = ""
        do {
            var request = authenticatedConsoleRequest(
                port: port,
                path: "/api/agents/\(agent.id.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? agent.id)",
                method: "PATCH"
            )
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONEncoder().encode(
                NativeUpdateAgentRequest(
                    human: NativeUpdateAgentHuman(
                        agentName: cleanName,
                        persona: agent.human?.persona ?? "",
                        tone: agent.human?.tone ?? "",
                        discussionMode: agent.human?.discussionMode ?? "",
                        memoryNotes: agent.human?.memoryNotes ?? "",
                        statusStyle: agent.human?.statusStyle ?? ""
                    )
                )
            )
            let (data, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                let message = String(data: data, encoding: .utf8) ?? "Could not save agent name."
                throw NSError(domain: "JameClaw", code: 1, userInfo: [NSLocalizedDescriptionKey: message])
            }
            await load()
            return true
        } catch {
            self.error = error.localizedDescription.isEmpty ? "Could not save agent name." : error.localizedDescription
            return false
        }
    }
}

private struct AgentManagerView: View {
    @StateObject private var store: NativeAgentStore
    private let port: Int
    @State private var selectedID = ""
    @State private var showingCreate = false
    @State private var showingTeamGrid = false
    @State private var creationMode: AgentCreationMode = .team

    init(port: Int) {
        self.port = port
        _store = StateObject(wrappedValue: NativeAgentStore(port: port))
    }

    private var selected: NativeAgentSummary? {
        store.agents.first(where: { $0.id == selectedID }) ?? store.agents.first
    }

    var body: some View {
        HStack(spacing: 0) {
            VStack(alignment: .leading, spacing: 0) {
                HStack {
                    Text("Agents").font(.title3.weight(.semibold))
                    Spacer()
                    Button { Task { await store.load() } } label: { Image(systemName: "arrow.clockwise") }
                        .help("Refresh agents")
                }
                .padding(16)
                Divider()

                if store.isLoading && store.agents.isEmpty {
                    ProgressView("Loading agents…").padding()
                } else if store.agents.isEmpty {
                    ContentUnavailableView("No agents found", systemImage: "sparkles")
                } else {
                    List(store.agents, selection: $selectedID) { agent in
                        HStack(spacing: 10) {
                            Image(systemName: agentIdentitySymbol(agent))
                                .foregroundStyle(agent.id == "main" ? JameBrand.orange : .blue)
                                .frame(width: 18)
                                .accessibilityLabel(agent.id == "main" ? "JameClaw icon" : "Unique \(agent.name.isEmpty ? agent.id : agent.name) icon")
                            VStack(alignment: .leading, spacing: 2) {
                                Text(agent.name.isEmpty ? agent.id : agent.name).font(.headline)
                                Text(agent.id).font(.caption.monospaced()).foregroundStyle(.secondary)
                            }
                        }
                        .tag(agent.id)
                    }
                }
            }
            .frame(minWidth: 180, idealWidth: 240, maxWidth: 300)

            Divider()

            Group {
                if let agent = selected {
                    AgentDetailView(
                        agent: agent,
                        rename: { name in Task { _ = await store.rename(agent, to: name) } },
                        addTeamAgent: {
                            creationMode = .team
                            showingCreate = true
                        },
                        spawnSubagent: {
                            creationMode = .subagent
                            showingCreate = true
                        },
                        showTeamGrid: {
                            showingTeamGrid = true
                        }
                    )
                    .id(agent.id)
                } else {
                    ContentUnavailableView("Choose an agent", systemImage: "person.3")
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .overlay(alignment: .bottomLeading) {
            if !store.error.isEmpty {
                Text(store.error)
                    .font(.caption)
                    .foregroundStyle(.red)
                    .padding(10)
            }
        }
        .task {
            await store.load()
            if selectedID.isEmpty { selectedID = store.agents.first?.id ?? "" }
        }
        .onReceive(NotificationCenter.default.publisher(for: .jameclawTeamGrid)) { _ in
            showingTeamGrid = true
        }
        .sheet(isPresented: $showingCreate) {
            CreateAgentView(mode: creationMode, parent: selected, store: store) { newID in
                selectedID = newID
                showingCreate = false
            }
        }
        .sheet(isPresented: $showingTeamGrid) {
            TeamGridView(store: store, port: port)
        }
    }
}

private struct AgentDetailView: View {
    let agent: NativeAgentSummary
    let rename: (String) -> Void
    let addTeamAgent: () -> Void
    let spawnSubagent: () -> Void
    let showTeamGrid: () -> Void
    @State private var displayName: String

    init(agent: NativeAgentSummary, rename: @escaping (String) -> Void, addTeamAgent: @escaping () -> Void, spawnSubagent: @escaping () -> Void, showTeamGrid: @escaping () -> Void) {
        self.agent = agent
        self.rename = rename
        self.addTeamAgent = addTeamAgent
        self.spawnSubagent = spawnSubagent
        self.showTeamGrid = showTeamGrid
        _displayName = State(initialValue: agent.name.isEmpty ? agent.id : agent.name)
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                VStack(alignment: .leading, spacing: 12) {
                    VStack(alignment: .leading, spacing: 5) {
                        HStack(spacing: 8) {
                            TextField("Agent name", text: $displayName)
                                .font(.title2.weight(.semibold))
                                .textFieldStyle(.plain)
                            if agent.isDefault { Text("Default").font(.caption.weight(.semibold)).padding(.horizontal, 7).padding(.vertical, 3).background(.yellow.opacity(0.2)).clipShape(Rectangle()) }
                        }
                        HStack(spacing: 8) {
                            Text(agent.id).font(.subheadline.monospaced()).foregroundStyle(.secondary)
                            Button("Save name") { rename(displayName) }
                                .controlSize(.small)
                                .disabled(displayName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || displayName == agent.name)
                        }
                    }
                    ScrollView(.horizontal) {
                        HStack(spacing: 8) {
                            Button("Add team agent", action: addTeamAgent)
                                .buttonStyle(.bordered)
                            Button("Spawn subagent", action: spawnSubagent)
                                .buttonStyle(.borderedProminent)
                            Button("Team grid", systemImage: "point.3.connected.trianglepath.dotted", action: showTeamGrid)
                                .buttonStyle(.bordered)
                                .help("See how team and spawned agents are connected")
                        }
                    }
                    .scrollIndicators(.hidden)
                }

                GroupBox("Configuration") {
                    Grid(alignment: .leading, horizontalSpacing: 18, verticalSpacing: 10) {
                        AgentField("Model", agent.model.isEmpty ? "Inherited default" : agent.model)
                        AgentField("Workspace", agent.workspace.isEmpty ? "Default workspace" : agent.workspace)
                        AgentField("Persona", agent.human?.persona?.isEmpty == false ? agent.human!.persona! : "Not set")
                        AgentField("Tone", agent.human?.tone?.isEmpty == false ? agent.human!.tone! : "Not set")
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }

                GroupBox("Capabilities") {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Skills: \((agent.skills ?? []).isEmpty ? "None configured" : (agent.skills ?? []).joined(separator: ", "))")
                        Text("Delegated agents: \((agent.subagents ?? []).isEmpty ? "None configured" : (agent.subagents ?? []).joined(separator: ", "))")
                    }
                    .font(.subheadline)
                    .frame(maxWidth: .infinity, alignment: .leading)
                }

                HStack(spacing: 28) {
                    AgentMetric("Sessions", agent.sessionCount ?? 0)
                    AgentMetric("Messages", agent.messageCount ?? 0)
                    AgentMetric("Tool calls", agent.toolCalls ?? 0)
                }
            }
            .padding(24)
        }
    }
}

private struct TeamGridActivityResponse: Codable {
    let files: [TeamGridFileAccess]
    let sources: [TeamGridInformationSource]
}

private struct TeamGridPermissionConfig: Codable {
    let agents: TeamGridPermissionAgents
    let tools: TeamGridPermissionTools
}

private struct TeamGridPermissionAgents: Codable {
    let defaults: TeamGridPermissionDefaults
}

private struct TeamGridPermissionDefaults: Codable {
    let workspace: String
    let restrictToWorkspace: Bool

    enum CodingKeys: String, CodingKey {
        case workspace
        case restrictToWorkspace = "restrict_to_workspace"
    }
}

private struct TeamGridPermissionTools: Codable {
    let allowWritePaths: [String]
    let writeFile: TeamGridToolPermission?
    let editFile: TeamGridToolPermission?
    let appendFile: TeamGridToolPermission?

    enum CodingKeys: String, CodingKey {
        case allowWritePaths = "allow_write_paths"
        case writeFile = "write_file"
        case editFile = "edit_file"
        case appendFile = "append_file"
    }
}

private struct TeamGridToolPermission: Codable {
    let enabled: Bool
}

private struct TeamGridArtifactList: Codable {
    let items: [TeamGridArtifact]
}

private struct TeamGridArtifact: Codable, Identifiable {
    let id: String
    let name: String
    let kind: String
    let updatedAtMS: Int64

    enum CodingKeys: String, CodingKey {
        case id, name, kind
        case updatedAtMS = "updated_at_ms"
    }
}

private struct TeamGridFileAccess: Codable, Identifiable {
    let path: String
    let accesses: Int
    let agents: [String]
    var id: String { path }
}

private struct TeamGridInformationSource: Codable, Identifiable {
    let id: String
    let name: String
    let sessions: Int
    let messages: Int
    let agents: [String]
}

private struct TeamGridInitiativeRecord: Codable, Identifiable {
    let checkedAt: String
    let status: String
    let summary: String
    var id: String { "\(checkedAt)-\(status)" }

    enum CodingKeys: String, CodingKey {
        case status, summary
        case checkedAt = "checked_at"
    }
}

private struct TeamGridInitiativeResponse: Codable {
    let enabled: Bool
    let initiative: Bool
    let intervalMinutes: Int
    let latest: TeamGridInitiativeRecord?
    let history: [TeamGridInitiativeRecord]

    enum CodingKeys: String, CodingKey {
        case enabled, initiative, latest, history
        case intervalMinutes = "interval_minutes"
    }
}

private struct TeamOperationsGoal: Codable, Identifiable {
    let id: String
    let title: String
    let outcome: String
    let leadAgentID: String
    let status: String

    enum CodingKeys: String, CodingKey {
        case id, title, outcome, status
        case leadAgentID = "lead_agent_id"
    }
}

private struct TeamOperationsTask: Codable, Identifiable {
    let id: String
    let goalID: String
    let kind: String?
    let title: String
    let description: String?
    let ownerAgentID: String?
    let status: String
    let dependsOn: [String]
    let acceptanceCriteria: [String]
    let fileScopes: [String]
    let timeBudgetMinutes: Int
    let tokenBudget: Int64
    let result: String?
    let verification: String?
    let blockedReason: String?

    enum CodingKeys: String, CodingKey {
        case id, kind, title, description, status, result, verification
        case goalID = "goal_id"
        case ownerAgentID = "owner_agent_id"
        case dependsOn = "depends_on"
        case acceptanceCriteria = "acceptance_criteria"
        case fileScopes = "file_scopes"
        case timeBudgetMinutes = "time_budget_minutes"
        case tokenBudget = "token_budget"
        case blockedReason = "blocked_reason"
    }

    var nodeKind: String { kind == "loop" ? "loop" : "task" }
}

private struct TeamOperationsSnapshot: Codable {
    let goal: TeamOperationsGoal?
    let tasks: [TeamOperationsTask]
    static let empty = TeamOperationsSnapshot(goal: nil, tasks: [])
}

private func teamPageBackground(_ scheme: ColorScheme) -> Color {
    scheme == .light ? Color(red: 0.965, green: 0.965, blue: 0.955) : JameBrand.ink
}

private func teamPanel(_ scheme: ColorScheme) -> Color {
    scheme == .light ? .white : JameBrand.panel
}

private func teamSurface(_ scheme: ColorScheme) -> Color {
    scheme == .light ? JameBrand.ink.opacity(0.035) : JameBrand.ink
}

private func teamPrimary(_ scheme: ColorScheme) -> Color {
    scheme == .light ? JameBrand.ink : JameBrand.paper
}

private func teamMuted(_ scheme: ColorScheme) -> Color {
    scheme == .light ? JameBrand.ink.opacity(0.58) : JameBrand.muted
}

private func teamRule(_ scheme: ColorScheme) -> Color {
    scheme == .light ? JameBrand.ink.opacity(0.14) : JameBrand.rule
}

@MainActor
private final class TeamGridActivityStore: ObservableObject {
    @Published var servers: [MCPServer] = []
    @Published var files: [TeamGridFileAccess] = []
    @Published var sources: [TeamGridInformationSource] = []
    @Published var initiative: TeamGridInitiativeResponse?
    @Published var operations = TeamOperationsSnapshot.empty
    @Published var permissions: TeamGridPermissionConfig?
    @Published var artifacts: [TeamGridArtifact] = []
    @Published var error = ""
    @Published var operationsStatus = ""

    func load(port: Int) async {
        error = ""
        do {
            async let serverResponse: (Data, URLResponse) = URLSession.shared.data(
                for: authenticatedConsoleRequest(port: port, path: "/api/tools/mcp/servers")
            )
            async let activityResponse: (Data, URLResponse) = URLSession.shared.data(
                for: authenticatedConsoleRequest(port: port, path: "/api/agents/activity-map")
            )
            async let initiativeResponse: (Data, URLResponse) = URLSession.shared.data(
                for: authenticatedConsoleRequest(port: port, path: "/api/agents/initiative")
            )
            async let operationsResponse: (Data, URLResponse) = URLSession.shared.data(
                for: authenticatedConsoleRequest(port: port, path: "/api/agents/team-operations")
            )
            async let permissionResponse: (Data, URLResponse) = URLSession.shared.data(
                for: authenticatedConsoleRequest(port: port, path: "/api/config")
            )
            async let artifactResponse: (Data, URLResponse) = URLSession.shared.data(
                for: authenticatedConsoleRequest(port: port, path: "/api/artifacts")
            )
            let (serverData, serverHTTP) = try await serverResponse
            let (activityData, activityHTTP) = try await activityResponse
            let (initiativeData, initiativeHTTP) = try await initiativeResponse
            let (operationsData, operationsHTTP) = try await operationsResponse
            let (permissionData, permissionHTTP) = try await permissionResponse
            let (artifactData, artifactHTTP) = try await artifactResponse
            guard let serverHTTP = serverHTTP as? HTTPURLResponse, (200..<300).contains(serverHTTP.statusCode),
                  let activityHTTP = activityHTTP as? HTTPURLResponse, (200..<300).contains(activityHTTP.statusCode),
                  let initiativeHTTP = initiativeHTTP as? HTTPURLResponse, (200..<300).contains(initiativeHTTP.statusCode),
                  let operationsHTTP = operationsHTTP as? HTTPURLResponse, (200..<300).contains(operationsHTTP.statusCode),
                  let permissionHTTP = permissionHTTP as? HTTPURLResponse, (200..<300).contains(permissionHTTP.statusCode),
                  let artifactHTTP = artifactHTTP as? HTTPURLResponse, (200..<300).contains(artifactHTTP.statusCode) else {
                throw NSError(domain: "JameClaw", code: 1, userInfo: [NSLocalizedDescriptionKey: "Could not load team-grid activity."])
            }
            servers = try JSONDecoder().decode(MCPServerList.self, from: serverData).servers
            let activity = try JSONDecoder().decode(TeamGridActivityResponse.self, from: activityData)
            files = activity.files
            sources = activity.sources
            initiative = try JSONDecoder().decode(TeamGridInitiativeResponse.self, from: initiativeData)
            operations = try JSONDecoder().decode(TeamOperationsSnapshot.self, from: operationsData)
            permissions = try JSONDecoder().decode(TeamGridPermissionConfig.self, from: permissionData)
            artifacts = try JSONDecoder().decode(TeamGridArtifactList.self, from: artifactData).items
        } catch {
            self.error = error.localizedDescription
        }
    }

    func saveGoal(port: Int, title: String, outcome: String, leadAgentID: String) async -> Bool {
        await sendOperation(
            port: port,
            path: "/api/agents/team-operations/goal",
            method: "PUT",
            payload: ["title": title, "outcome": outcome, "lead_agent_id": leadAgentID],
            success: "Team goal saved."
        )
    }

    func createTask(
        port: Int,
        kind: String,
        title: String,
        description: String,
        ownerAgentID: String,
        dependsOn: [String],
        acceptanceCriteria: [String],
        fileScopes: [String],
        timeBudgetMinutes: Int,
        tokenBudget: Int
    ) async -> Bool {
        await sendOperation(
            port: port,
            path: "/api/agents/team-operations/tasks",
            method: "POST",
            payload: [
                "kind": kind,
                "title": title,
                "description": description,
                "owner_agent_id": ownerAgentID,
                "depends_on": dependsOn,
                "acceptance_criteria": acceptanceCriteria,
                "file_scopes": fileScopes,
                "time_budget_minutes": timeBudgetMinutes,
                "token_budget": tokenBudget,
            ],
            success: kind == "loop" ? "Loop node created. The next step is gated until it is verified." : "Team task created."
        )
    }

    func updateTask(
        port: Int,
        task: TeamOperationsTask,
        kind: String,
        title: String,
        description: String,
        ownerAgentID: String,
        dependsOn: [String],
        acceptanceCriteria: [String],
        fileScopes: [String],
        timeBudgetMinutes: Int,
        tokenBudget: Int
    ) async -> Bool {
        await sendOperation(
            port: port,
            path: "/api/agents/team-operations/tasks/\(task.id)",
            method: "PATCH",
            payload: [
                "kind": kind,
                "title": title,
                "description": description,
                "owner_agent_id": ownerAgentID,
                "depends_on": dependsOn,
                "acceptance_criteria": acceptanceCriteria,
                "file_scopes": fileScopes,
                "time_budget_minutes": timeBudgetMinutes,
                "token_budget": tokenBudget,
            ],
            success: "Task contract updated."
        )
    }

    func taskAction(port: Int, task: TeamOperationsTask, action: String, detail: String = "") async -> Bool {
        var payload: [String: Any] = ["action": action]
        switch action {
        case "submit_review": payload["result"] = detail
        case "complete": payload["verification"] = detail
        case "block": payload["blocked_reason"] = detail
        default: break
        }
        return await sendOperation(
            port: port,
            path: "/api/agents/team-operations/tasks/\(task.id)/action",
            method: "POST",
            payload: payload,
            success: "Task moved to \(action.replacingOccurrences(of: "_", with: " "))."
        )
    }

    private func sendOperation(port: Int, path: String, method: String, payload: [String: Any], success: String) async -> Bool {
        do {
            var request = authenticatedConsoleRequest(port: port, path: path, method: method)
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: payload)
            let (data, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                let message = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)
                operationsStatus = message?.isEmpty == false ? message! : "Team operation was rejected."
                return false
            }
            operationsStatus = success
            await load(port: port)
            return true
        } catch {
            operationsStatus = error.localizedDescription
            return false
        }
    }
}

private struct TeamGridView: View {
    @ObservedObject var store: NativeAgentStore
    let port: Int
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var activity = TeamGridActivityStore()
    @StateObject private var providers = NativeProviderStore()
    @State private var selectedAgentID = "main"
    @State private var showingCreate = false
    @State private var creationMode: AgentCreationMode = .subagent
    @State private var showingGoalEditor = false
    @State private var showingTaskEditor = false
    @State private var editingTask: TeamOperationsTask?
    @State private var preferredTaskOwnerID = ""
    @State private var preferredNodeKind = "task"
    @State private var taskActionRequest: TeamTaskActionRequest?
    @AppStorage("launcher.teamGrid.zoom") private var gridZoom = 1.0

    private var agents: [NativeAgentSummary] { store.agents }

    private var agentsByID: [String: NativeAgentSummary] {
        Dictionary(uniqueKeysWithValues: agents.map { ($0.id, $0) })
    }
    private var mainAgent: NativeAgentSummary? { agentsByID["main"] ?? agents.first(where: \ .isDefault) }
    private var selectedAgent: NativeAgentSummary? {
        agentsByID[selectedAgentID] ?? mainAgent ?? agents.first
    }
    private var spawnedIDs: Set<String> {
        Set(agents.flatMap { $0.subagents ?? [] })
    }
    private var independentTeam: [NativeAgentSummary] {
        agents.filter { $0.id != "main" && !spawnedIDs.contains($0.id) }
    }

    private func modelLabel(for agent: NativeAgentSummary) -> String {
        let identities = [agent.id, agent.name, agent.human?.agentName ?? ""]
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() }
        if identities.contains("hermes") {
            // Hermes is an external Nous runtime. A missing JameClaw override
            // does not mean it inherited the global Codex CLI model.
            let runtime = store.localAgents.first(where: { $0.preset == .hermes })
            let provider = runtime?.runtimeProvider?.trimmingCharacters(in: .whitespacesAndNewlines)
            let model = runtime?.runtimeModel?.trimmingCharacters(in: .whitespacesAndNewlines)
            return "\((provider?.isEmpty == false ? provider! : "Nous Research")) · \((model?.isEmpty == false ? model! : "Model not reported"))"
        }
        let configuredModel = agent.model.trimmingCharacters(in: .whitespacesAndNewlines)
        let modelName = configuredModel.isEmpty ? providers.defaultModel : configuredModel
        guard !modelName.isEmpty else { return "Default model (not reported)" }
        let inheritedSuffix = configuredModel.isEmpty ? " · inherited" : ""
        if let model = providers.models.first(where: { $0.modelName == modelName }) {
            return "\(providers.providerName(for: model)) · \(modelName)\(inheritedSuffix)"
        }
        return modelName + inheritedSuffix
    }

    private func activeTask(for agent: NativeAgentSummary) -> TeamOperationsTask? {
        let priority = ["working", "review", "planned", "blocked", "paused", "unassigned"]
        return activity.operations.tasks
            .filter { $0.ownerAgentID == agent.id && $0.status != "done" }
            .sorted {
                (priority.firstIndex(of: $0.status) ?? priority.count)
                    < (priority.firstIndex(of: $1.status) ?? priority.count)
            }
            .first
    }

    private var delegatableTasks: [TeamOperationsTask] {
        activity.operations.tasks.filter { !["working", "review", "done"].contains($0.status) }
    }

    private func delegate(_ task: TeamOperationsTask, to agent: NativeAgentSummary) {
        Task {
            _ = await activity.updateTask(
                port: port,
                task: task,
                kind: task.nodeKind,
                title: task.title,
                description: task.description ?? "",
                ownerAgentID: agent.id,
                dependsOn: task.dependsOn,
                acceptanceCriteria: task.acceptanceCriteria,
                fileScopes: task.fileScopes,
                timeBudgetMinutes: task.timeBudgetMinutes,
                tokenBudget: Int(task.tokenBudget)
            )
        }
    }

    private func createTask(for agent: NativeAgentSummary) {
        selectedAgentID = agent.id
        preferredTaskOwnerID = agent.id
        preferredNodeKind = "task"
        editingTask = nil
        showingTaskEditor = true
    }

    private func createLoop(for agent: NativeAgentSummary? = nil) {
        if let agent { selectedAgentID = agent.id }
        preferredTaskOwnerID = agent?.id ?? ""
        preferredNodeKind = "loop"
        editingTask = nil
        showingTaskEditor = true
    }

    var body: some View {
        VStack(spacing: 0) {
            VStack(alignment: .leading, spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    Text("TEAM OPERATIONS")
                        .font(.system(size: 10, weight: .semibold, design: .rounded))
                        .tracking(1.4).foregroundStyle(JameBrand.orange)
                    Text("Team grid").font(.system(size: 27, weight: .semibold, design: .rounded))
                    Text("Inspect activity, move through the hierarchy, and create agents without leaving the map.")
                        .font(.caption)
                        .foregroundStyle(teamMuted(colorScheme))
                }
                ScrollView(.horizontal) {
                    HStack(spacing: 8) {
                        Button { Task { await refreshGrid() } } label: { Image(systemName: "arrow.clockwise") }
                            .buttonStyle(.bordered).help("Refresh agents and activity")
                        HStack(spacing: 4) {
                            Button {
                                gridZoom = max(0.6, ((gridZoom - 0.1) * 10).rounded() / 10)
                            } label: {
                                Image(systemName: "minus")
                            }
                            .disabled(gridZoom <= 0.6)
                            .help("Minimize Team Grid")
                            Text("\(Int((gridZoom * 100).rounded()))%")
                                .font(.caption.monospaced().weight(.semibold))
                                .frame(width: 42)
                            Button {
                                gridZoom = min(1.4, ((gridZoom + 0.1) * 10).rounded() / 10)
                            } label: {
                                Image(systemName: "plus")
                            }
                            .disabled(gridZoom >= 1.4)
                            .help("Enlarge Team Grid")
                        }
                        .buttonStyle(.bordered)
                        Button(activity.operations.goal == nil ? "Set team goal" : "Edit goal") {
                            showingGoalEditor = true
                        }
                        .buttonStyle(.bordered)
                        Button("Add task") {
                            preferredTaskOwnerID = ""
                            preferredNodeKind = "task"
                            editingTask = nil
                            showingTaskEditor = true
                        }
                        .buttonStyle(.bordered)
                        .disabled(activity.operations.goal == nil)
                        Button("Add loop", systemImage: "repeat") {
                            createLoop()
                        }
                        .buttonStyle(.bordered)
                        .disabled(activity.operations.goal == nil)
                        Button("Add team agent") {
                            creationMode = .team
                            showingCreate = true
                        }
                        .buttonStyle(.bordered)
                        Button("Spawn subagent") {
                            creationMode = .subagent
                            showingCreate = true
                        }
                        .buttonStyle(.borderedProminent).tint(JameBrand.orange)
                        Button("Done") { dismiss() }.buttonStyle(.bordered)
                    }
                }
                .scrollIndicators(.hidden)
            }
            .padding(22)
            Rectangle().fill(teamRule(colorScheme)).frame(height: 1)

            HStack(spacing: 0) {
                // Keep the axes independent so a trackpad or mouse wheel can
                // move vertically without accidentally locking onto the wide
                // team map. The inner horizontal scroller handles zoomed maps.
                ScrollView(.vertical) {
                    ScrollView(.horizontal) {
                        TeamGridZoomContainer(scale: gridZoom) {
                        VStack(spacing: 28) {
                        TeamOperationsBoard(
                            snapshot: activity.operations,
                            agents: agents,
                            editTask: { task in
                                editingTask = task
                                preferredNodeKind = task.nodeKind
                                showingTaskEditor = true
                            },
                            act: { task, action in
                                if action == "start" || action == "pause" || action == "reopen" {
                                    Task { _ = await activity.taskAction(port: port, task: task, action: action) }
                                } else {
                                    taskActionRequest = TeamTaskActionRequest(task: task, action: action)
                                }
                            }
                        )
                        TeamGridFlowLine(length: 24)
                        if let initiative = activity.initiative {
                            TeamGridInitiativeSection(initiative: initiative)
                            TeamGridFlowLine(length: 24)
                        }
                        if let mainAgent {
                            if !activity.sources.isEmpty {
                                TeamGridInformationSources(sources: activity.sources)
                                TeamGridFlowLine(length: 30)
                            }
                            AgentTreeBranch(
                                agent: mainAgent,
                                agentsByID: agentsByID,
                                visited: [],
                                selectedAgentID: selectedAgentID,
                                modelLabel: modelLabel,
                                activeTask: activeTask,
                                delegatableTasks: delegatableTasks,
                                delegateTask: delegate,
                                createTask: createTask,
                                createLoop: { createLoop(for: $0) },
                                select: { selectedAgentID = $0 }
                            )
                        } else {
                            ContentUnavailableView("No main agent", systemImage: "person.3")
                        }

                        TeamGridMCPSection(servers: activity.servers)

                        if !independentTeam.isEmpty {
                            VStack(spacing: 14) {
                                Label("Independent team agents", systemImage: "person.3.fill")
                                    .font(.headline)
                                    .foregroundStyle(teamPrimary(colorScheme))
                                HStack(alignment: .top, spacing: 16) {
                                    ForEach(independentTeam) { agent in
                                        TeamGridNode(
                                            agent: agent,
                                            kind: .team,
                                            isSelected: selectedAgentID == agent.id,
                                            modelLabel: modelLabel(for: agent),
                                            activeTask: activeTask(for: agent),
                                            delegatableTasks: delegatableTasks,
                                            delegateTask: { delegate($0, to: agent) },
                                            createTask: { createTask(for: agent) },
                                            createLoop: { createLoop(for: agent) },
                                            select: { selectedAgentID = agent.id }
                                        )
                                    }
                                }
                            }
                            .padding(.top, 8)
                        }

                        if let permissions = activity.permissions {
                            TeamGridModificationSection(
                                permissions: permissions,
                                artifacts: activity.artifacts
                            )
                        }

                        TeamGridFileSection(files: activity.files)

                        if !activity.error.isEmpty {
                            Label(activity.error, systemImage: "exclamationmark.triangle")
                                .font(.caption)
                                .foregroundStyle(JameBrand.orange)
                        }
                        if !activity.operationsStatus.isEmpty {
                            Text(activity.operationsStatus)
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(JameBrand.orange)
                        }
                    }
                        .padding(36)
                        .frame(minWidth: 440, minHeight: 400)
                        }
                    }
                    .scrollIndicators(.visible)
                }
                .scrollIndicators(.visible)
                .frame(maxWidth: .infinity, maxHeight: .infinity)

                Rectangle().fill(teamRule(colorScheme)).frame(width: 1)

                TeamGridInspector(
                    agent: selectedAgent,
                    modelLabel: selectedAgent.map(modelLabel) ?? "Model unavailable",
                    tasks: selectedAgent.map { selected in
                        activity.operations.tasks.filter { $0.ownerAgentID == selected.id }
                    } ?? [],
                    sources: activity.sources,
                    files: activity.files,
                    initiative: activity.initiative,
                    permissions: activity.permissions,
                    spawnSubagent: {
                        creationMode = .subagent
                        showingCreate = true
                    },
                    addTeamAgent: {
                        creationMode = .team
                        showingCreate = true
                    }
                )
                .frame(minWidth: 220, idealWidth: 250, maxWidth: 280)
                .frame(maxHeight: .infinity)
                .background(teamPanel(colorScheme))
            }
        }
        .foregroundStyle(teamPrimary(colorScheme))
        .background(teamPageBackground(colorScheme))
        .frame(minWidth: 760, idealWidth: 1240, minHeight: 500)
        .task {
            await refreshGrid()
            if agentsByID[selectedAgentID] == nil { selectedAgentID = mainAgent?.id ?? agents.first?.id ?? "" }
        }
        .sheet(isPresented: $showingCreate) {
            CreateAgentView(mode: creationMode, parent: selectedAgent ?? mainAgent, store: store) { newID in
                selectedAgentID = newID
                showingCreate = false
                Task { await refreshGrid() }
            }
        }
        .sheet(isPresented: $showingGoalEditor) {
            TeamGoalEditor(
                goal: activity.operations.goal,
                agents: agents,
                save: { title, outcome, leadAgentID in
                    let saved = await activity.saveGoal(port: port, title: title, outcome: outcome, leadAgentID: leadAgentID)
                    if saved { showingGoalEditor = false }
                }
            )
        }
        .sheet(isPresented: $showingTaskEditor) {
            TeamTaskEditor(
                task: editingTask,
                tasks: activity.operations.tasks,
                agents: agents,
                defaultOwnerAgentID: preferredTaskOwnerID,
                defaultNodeKind: preferredNodeKind,
                save: { kind, title, description, owner, dependencies, criteria, scopes, minutes, tokens in
                    let saved: Bool
                    if let editingTask {
                        saved = await activity.updateTask(
                            port: port, task: editingTask, kind: kind, title: title, description: description,
                            ownerAgentID: owner, dependsOn: dependencies, acceptanceCriteria: criteria,
                            fileScopes: scopes, timeBudgetMinutes: minutes, tokenBudget: tokens
                        )
                    } else {
                        saved = await activity.createTask(
                            port: port, kind: kind, title: title, description: description,
                            ownerAgentID: owner, dependsOn: dependencies, acceptanceCriteria: criteria,
                            fileScopes: scopes, timeBudgetMinutes: minutes, tokenBudget: tokens
                        )
                    }
                    if saved { showingTaskEditor = false }
                }
            )
        }
        .sheet(item: $taskActionRequest) { request in
            TeamTaskEvidenceEditor(request: request) { detail in
                let saved = await activity.taskAction(port: port, task: request.task, action: request.action, detail: detail)
                if saved { taskActionRequest = nil }
            }
        }
    }

    private func refreshGrid() async {
        async let agentsLoad: Void = store.load()
        async let activityLoad: Void = activity.load(port: port)
        async let providersLoad: Void = providers.load(port: port)
        _ = await (agentsLoad, activityLoad, providersLoad)
    }
}

private struct TeamTaskActionRequest: Identifiable {
    let task: TeamOperationsTask
    let action: String
    var id: String { "\(task.id)-\(action)" }
}

private struct TeamOperationsBoard: View {
    let snapshot: TeamOperationsSnapshot
    let agents: [NativeAgentSummary]
    let editTask: (TeamOperationsTask) -> Void
    let act: (TeamOperationsTask, String) -> Void
    @Environment(\.colorScheme) private var colorScheme

    private var panel: Color { colorScheme == .light ? .white : JameBrand.panel }
    private var primary: Color { colorScheme == .light ? JameBrand.ink : JameBrand.paper }
    private var muted: Color { colorScheme == .light ? JameBrand.ink.opacity(0.58) : JameBrand.muted }
    private var rule: Color { colorScheme == .light ? JameBrand.ink.opacity(0.14) : JameBrand.rule }
    private var agentNames: [String: String] {
        Dictionary(uniqueKeysWithValues: agents.map { ($0.id, $0.name.isEmpty ? $0.id : $0.name) })
    }
    private var taskTitles: [String: String] {
        Dictionary(uniqueKeysWithValues: snapshot.tasks.map { ($0.id, $0.title) })
    }
    private let lanes: [(String, [String])] = [
        ("BACKLOG", ["unassigned", "planned"]),
        ("WORKING", ["working"]),
        ("REVIEW", ["review"]),
        ("BLOCKED", ["blocked", "paused"]),
        ("DONE", ["done"]),
    ]

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 4) {
                    Text("TEAM LEAD")
                        .font(.system(size: 9, weight: .semibold, design: .rounded))
                        .tracking(1.1).foregroundStyle(JameBrand.orange)
                    if let goal = snapshot.goal {
                        Text(goal.title).font(.system(size: 20, weight: .semibold, design: .rounded)).foregroundStyle(primary)
                        Text(goal.outcome).font(.caption).foregroundStyle(muted).lineLimit(3)
                    } else {
                        Text("No operating goal yet").font(.headline).foregroundStyle(primary)
                        Text("Set a measurable goal, then assign dependency-aware tasks to the team.")
                            .font(.caption).foregroundStyle(muted)
                    }
                }
                Spacer()
                if let goal = snapshot.goal {
                    VStack(alignment: .trailing, spacing: 4) {
                        Text(goal.status.uppercased()).font(.system(size: 9, weight: .semibold, design: .rounded))
                            .foregroundStyle(JameBrand.orange)
                        Text("Lead · \(agentNames[goal.leadAgentID] ?? goal.leadAgentID)")
                            .font(.caption.weight(.semibold)).foregroundStyle(primary)
                        Text("\(snapshot.tasks.filter { $0.status == "done" }.count)/\(snapshot.tasks.count) nodes verified")
                            .font(.caption2).foregroundStyle(muted)
                    }
                }
            }

            if snapshot.goal != nil {
                HStack(alignment: .top, spacing: 10) {
                    ForEach(lanes, id: \.0) { lane in
                        TeamTaskLane(
                            title: lane.0,
                            tasks: snapshot.tasks.filter { lane.1.contains($0.status) },
                            agentNames: agentNames,
                            taskTitles: taskTitles,
                            editTask: editTask,
                            act: act
                        )
                    }
                }
            }
        }
        .padding(16)
        .frame(width: 1110, alignment: .leading)
        .background(panel)
        .overlay(Rectangle().stroke(rule))
        .overlay(alignment: .leading) { Rectangle().fill(JameBrand.orange).frame(width: 3) }
    }
}

private struct TeamTaskLane: View {
    let title: String
    let tasks: [TeamOperationsTask]
    let agentNames: [String: String]
    let taskTitles: [String: String]
    let editTask: (TeamOperationsTask) -> Void
    let act: (TeamOperationsTask, String) -> Void
    @Environment(\.colorScheme) private var colorScheme

    private var surface: Color { colorScheme == .light ? JameBrand.ink.opacity(0.035) : JameBrand.ink }
    private var primary: Color { colorScheme == .light ? JameBrand.ink : JameBrand.paper }
    private var muted: Color { colorScheme == .light ? JameBrand.ink.opacity(0.55) : JameBrand.muted }
    private var rule: Color { colorScheme == .light ? JameBrand.ink.opacity(0.13) : JameBrand.rule }

    var body: some View {
        VStack(alignment: .leading, spacing: 9) {
            HStack {
                Text(title).font(.system(size: 9, weight: .semibold, design: .rounded)).tracking(0.9).foregroundStyle(muted)
                Spacer()
                Text("\(tasks.count)").font(.caption2.monospaced()).foregroundStyle(JameBrand.orange)
            }
            if tasks.isEmpty {
                Text("No tasks").font(.caption2).foregroundStyle(muted).padding(.vertical, 8)
            } else {
                ForEach(tasks) { task in
                    VStack(alignment: .leading, spacing: 7) {
                        if task.nodeKind == "loop" {
                            Label("LOOP GATE", systemImage: "repeat")
                                .font(.system(size: 9, weight: .bold, design: .rounded))
                                .tracking(0.7)
                                .foregroundStyle(JameBrand.orange)
                        }
                        Text(task.title).font(.subheadline.weight(.semibold)).foregroundStyle(primary).lineLimit(2)
                        Label(agentNames[task.ownerAgentID ?? ""] ?? "Unassigned", systemImage: "person.fill")
                            .font(.caption2).foregroundStyle(muted)
                        if !task.dependsOn.isEmpty {
                            Text("After: " + task.dependsOn.compactMap { taskTitles[$0] }.joined(separator: ", "))
                                .font(.caption2).foregroundStyle(muted).lineLimit(2)
                        }
                        if let reason = task.blockedReason, !reason.isEmpty {
                            Text(reason).font(.caption2).foregroundStyle(.red).lineLimit(2)
                        }
                        HStack(spacing: 5) {
                            if task.timeBudgetMinutes > 0 {
                                Text("\(task.timeBudgetMinutes)m").font(.caption2.monospaced()).foregroundStyle(muted)
                            }
                            if task.tokenBudget > 0 {
                                Text("\(task.tokenBudget) tok").font(.caption2.monospaced()).foregroundStyle(muted)
                            }
                            Spacer()
                        }
                        taskActions(task)
                    }
                    .padding(10)
                    .background(colorScheme == .light ? Color.white : JameBrand.panel)
                    .overlay(Rectangle().stroke(task.status == "working" ? JameBrand.orange : rule))
                }
            }
            Spacer(minLength: 0)
        }
        .padding(10)
        .frame(width: 204, alignment: .topLeading)
        .frame(minHeight: 160, alignment: .topLeading)
        .background(surface)
        .overlay(Rectangle().stroke(rule))
    }

    @ViewBuilder
    private func taskActions(_ task: TeamOperationsTask) -> some View {
        HStack(spacing: 6) {
            switch task.status {
            case "unassigned":
                Button("Assign") { editTask(task) }
            case "planned", "blocked", "paused":
                Button(task.nodeKind == "loop" ? "Run loop" : "Start") { act(task, "start") }
            case "working":
                Button(task.nodeKind == "loop" ? "Review loop" : "Review") { act(task, "submit_review") }
            case "review":
                Button("Verify") { act(task, "complete") }
            case "done":
                Button("Reopen") { act(task, "reopen") }
            default:
                EmptyView()
            }
            if task.status != "done" {
                Menu {
                    if task.status != "working" && task.status != "review" { Button("Edit contract") { editTask(task) } }
                    Button("Mark blocked") { act(task, "block") }
                    Button("Pause") { act(task, "pause") }
                } label: {
                    Image(systemName: "ellipsis")
                }
                .menuStyle(.borderlessButton)
            }
        }
        .buttonStyle(.bordered)
        .controlSize(.small)
    }
}

private struct TeamGoalEditor: View {
    let goal: TeamOperationsGoal?
    let agents: [NativeAgentSummary]
    let save: (String, String, String) async -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var title: String
    @State private var outcome: String
    @State private var leadAgentID: String
    @State private var saving = false

    init(goal: TeamOperationsGoal?, agents: [NativeAgentSummary], save: @escaping (String, String, String) async -> Void) {
        self.goal = goal
        self.agents = agents
        self.save = save
        _title = State(initialValue: goal?.title ?? "")
        _outcome = State(initialValue: goal?.outcome ?? "")
        _leadAgentID = State(initialValue: goal?.leadAgentID ?? "main")
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text(goal == nil ? "Set team goal" : "Edit team goal").font(.title2.weight(.semibold))
            Text("The Team Lead owns this outcome and coordinates every dependent task.").font(.caption).foregroundStyle(.secondary)
            TextField("Goal", text: $title)
            TextField("Measurable outcome", text: $outcome, axis: .vertical).lineLimit(3...6)
            Picker("Team Lead", selection: $leadAgentID) {
                ForEach(agents) { agent in Text(agent.name.isEmpty ? agent.id : agent.name).tag(agent.id) }
            }
            HStack {
                Spacer()
                Button("Cancel") { dismiss() }
                Button(saving ? "Saving…" : "Save goal") {
                    saving = true
                    Task { await save(title, outcome, leadAgentID); saving = false }
                }
                .buttonStyle(.borderedProminent).tint(JameBrand.orange)
                .disabled(saving || title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || outcome.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
        .padding(24).frame(width: 520)
    }
}

private struct TeamTaskEditor: View {
    let task: TeamOperationsTask?
    let tasks: [TeamOperationsTask]
    let agents: [NativeAgentSummary]
    let save: (String, String, String, String, [String], [String], [String], Int, Int) async -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var title: String
    @State private var taskDescription: String
    @State private var nodeKind: String
    @State private var ownerAgentID: String
    @State private var dependencyIDs: Set<String>
    @State private var criteriaText: String
    @State private var scopesText: String
    @State private var timeBudgetMinutes: Int
    @State private var tokenBudget: Int
    @State private var saving = false

    init(
        task: TeamOperationsTask?,
        tasks: [TeamOperationsTask],
        agents: [NativeAgentSummary],
        defaultOwnerAgentID: String = "",
        defaultNodeKind: String = "task",
        save: @escaping (String, String, String, String, [String], [String], [String], Int, Int) async -> Void
    ) {
        self.task = task
        self.tasks = tasks
        self.agents = agents
        self.save = save
        _title = State(initialValue: task?.title ?? "")
        _taskDescription = State(initialValue: task?.description ?? "")
        _nodeKind = State(initialValue: task?.nodeKind ?? defaultNodeKind)
        _ownerAgentID = State(initialValue: task?.ownerAgentID ?? defaultOwnerAgentID)
        _dependencyIDs = State(initialValue: Set(task?.dependsOn ?? []))
        _criteriaText = State(initialValue: (task?.acceptanceCriteria ?? []).joined(separator: "\n"))
        _scopesText = State(initialValue: (task?.fileScopes ?? []).joined(separator: "\n"))
        _timeBudgetMinutes = State(initialValue: task?.timeBudgetMinutes ?? 30)
        _tokenBudget = State(initialValue: Int(task?.tokenBudget ?? 8000))
    }

    private var dependencyOptions: [TeamOperationsTask] { tasks.filter { $0.id != task?.id } }
    private func lines(_ value: String) -> [String] {
        value.split(separator: "\n").map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }.filter { !$0.isEmpty }
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 15) {
                HStack {
                    Text(task == nil ? (nodeKind == "loop" ? "Create loop node" : "Create team task") : "Edit node contract")
                        .font(.title2.weight(.semibold))
                    if nodeKind == "loop" {
                        Label("LOOP", systemImage: "repeat")
                            .font(.caption.weight(.bold))
                            .foregroundStyle(JameBrand.orange)
                    }
                }
                Text(nodeKind == "loop"
                     ? "This checkpoint repeats until its evidence is verified. Every next node is automatically blocked behind it."
                     : "Define one owner, completion evidence, dependencies, scope, and budget before work starts.")
                    .font(.caption).foregroundStyle(.secondary)
                TextField("Task title", text: $title)
                TextField("Deliverable", text: $taskDescription, axis: .vertical).lineLimit(2...5)
                Picker("Owner", selection: $ownerAgentID) {
                    Text("Unassigned").tag("")
                    ForEach(agents) { agent in Text(agent.name.isEmpty ? agent.id : agent.name).tag(agent.id) }
                }
                if !dependencyOptions.isEmpty {
                    VStack(alignment: .leading, spacing: 7) {
                        Text("DEPENDENCIES").font(.caption.weight(.semibold)).foregroundStyle(JameBrand.orange)
                        ForEach(dependencyOptions) { dependency in
                            Toggle(dependency.title, isOn: Binding(
                                get: { dependencyIDs.contains(dependency.id) },
                                set: { enabled in
                                    if enabled { dependencyIDs.insert(dependency.id) } else { dependencyIDs.remove(dependency.id) }
                                }
                            ))
                        }
                    }
                }
                TextField("Acceptance criteria — one per line", text: $criteriaText, axis: .vertical).lineLimit(3...7)
                TextField("File scopes — one path per line", text: $scopesText, axis: .vertical).lineLimit(2...5)
                Stepper("Time budget: \(timeBudgetMinutes) minutes", value: $timeBudgetMinutes, in: 0...1440, step: 15)
                TextField("Token budget", value: $tokenBudget, format: .number)
                HStack {
                    Spacer()
                    Button("Cancel") { dismiss() }
                    Button(saving ? "Saving…" : (task == nil ? (nodeKind == "loop" ? "Create loop" : "Create task") : "Save contract")) {
                        saving = true
                        Task {
                            await save(nodeKind, title, taskDescription, ownerAgentID, Array(dependencyIDs), lines(criteriaText), lines(scopesText), timeBudgetMinutes, tokenBudget)
                            saving = false
                        }
                    }
                    .buttonStyle(.borderedProminent).tint(JameBrand.orange)
                    .disabled(saving || title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
            .padding(24)
        }
        .frame(width: 560, height: 650)
    }
}

private struct TeamTaskEvidenceEditor: View {
    let request: TeamTaskActionRequest
    let save: (String) async -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme
    @State private var detail = ""
    @State private var saving = false

    private var title: String {
        switch request.action {
        case "submit_review": return "Submit result for review"
        case "complete": return "Verify and complete"
        case "block": return "Mark task blocked"
        default: return "Update task"
        }
    }
    private var prompt: String {
        switch request.action {
        case "submit_review": return "Describe the concrete deliverable and where the reviewer can find it."
        case "complete": return "Provide test output, review evidence, or another concrete verification."
        case "block": return "Explain the blocker and what is needed to continue."
        default: return "Add evidence."
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text(title).font(.title2.weight(.semibold))
            Text(request.task.title).font(.headline).foregroundStyle(JameBrand.orange)
            Text(prompt).font(.caption).foregroundStyle(.secondary)
            TextEditor(text: $detail).frame(minHeight: 150).overlay(Rectangle().stroke(teamRule(colorScheme)))
            HStack {
                Spacer()
                Button("Cancel") { dismiss() }
                Button(saving ? "Saving…" : "Save") {
                    saving = true
                    Task { await save(detail); saving = false }
                }
                .buttonStyle(.borderedProminent).tint(JameBrand.orange)
                .disabled(saving || detail.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
        .padding(24).frame(width: 500)
    }
}

private struct TeamGridInitiativeSection: View {
    let initiative: TeamGridInitiativeResponse
    @Environment(\.colorScheme) private var colorScheme

    private var statusLabel: String {
        guard initiative.enabled else { return "Heartbeat off" }
        guard initiative.initiative else { return "Task-list mode" }
        switch initiative.latest?.status {
        case "working": return "Working"
        case "needs_approval": return "Needs approval"
        case "error": return "Check failed"
        case "completed": return "Problem solved"
        case "idle": return "Monitoring"
        default: return "Initiative ready"
        }
    }

    private var statusIcon: String {
        switch initiative.latest?.status {
        case "working": return "bolt.fill"
        case "needs_approval": return "hand.raised.fill"
        case "error": return "exclamationmark.triangle.fill"
        case "completed": return "checkmark.circle.fill"
        default: return "sparkles"
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Label("Agent initiative", systemImage: statusIcon)
                    .font(.headline)
                    .foregroundStyle(JameBrand.orange)
                Spacer()
                Text(statusLabel.uppercased())
                    .font(.system(size: 9, weight: .semibold, design: .rounded))
                    .tracking(0.8)
                    .foregroundStyle(initiative.enabled && initiative.initiative ? JameBrand.ink : teamPrimary(colorScheme))
                    .padding(.horizontal, 9).padding(.vertical, 5)
                    .background(
                        initiative.enabled && initiative.initiative ? JameBrand.orange : teamSurface(colorScheme),
                        in: Rectangle()
                    )
            }
            Text("Jame checks every \(initiative.intervalMinutes) minutes, finds one evidence-backed problem, and can solve safe local work without waiting for a prompt.")
                .font(.caption)
                .foregroundStyle(teamMuted(colorScheme))
            if let latest = initiative.latest {
                Text(latest.summary.isEmpty ? "Latest check completed without a report." : latest.summary)
                    .font(.system(size: 12, design: .rounded))
                    .foregroundStyle(teamPrimary(colorScheme))
                    .lineLimit(6)
                    .textSelection(.enabled)
                    .padding(12)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(teamSurface(colorScheme), in: Rectangle())
            } else {
                Text("Waiting for the first initiative check. The first one runs shortly after the gateway starts.")
                    .font(.caption)
                    .foregroundStyle(teamMuted(colorScheme))
            }
        }
        .padding(16)
        .frame(width: 540, alignment: .leading)
        .background(teamPanel(colorScheme), in: Rectangle())
        .overlay(Rectangle().stroke(JameBrand.orange.opacity(0.45)))
        .accessibilityElement(children: .combine)
    }
}

private struct TeamGridFlowLine: View {
    let length: CGFloat
    @State private var moving = false

    var body: some View {
        ZStack(alignment: .top) {
            Rectangle().fill(JameBrand.orange.opacity(0.28)).frame(width: 2, height: length)
            Circle()
                .fill(JameBrand.orange)
                .frame(width: 7, height: 7)
                .shadow(color: JameBrand.orange.opacity(0.7), radius: 5)
                .offset(y: moving ? max(0, length - 7) : 0)
        }
        .frame(width: 12, height: length)
        .onAppear {
            withAnimation(.linear(duration: 1.35).repeatForever(autoreverses: false)) {
                moving = true
            }
        }
        .accessibilityHidden(true)
    }
}

private struct TeamGridContentSizeKey: PreferenceKey {
    static var defaultValue: CGSize = .zero
    static func reduce(value: inout CGSize, nextValue: () -> CGSize) {
        let next = nextValue()
        if next.width > 0, next.height > 0 { value = next }
    }
}

private struct TeamGridZoomContainer<Content: View>: View {
    let scale: Double
    @ViewBuilder let content: () -> Content
    @State private var contentSize = CGSize(width: 440, height: 400)

    var body: some View {
        content()
            .background {
                GeometryReader { proxy in
                    Color.clear.preference(key: TeamGridContentSizeKey.self, value: proxy.size)
                }
            }
            .onPreferenceChange(TeamGridContentSizeKey.self) { contentSize = $0 }
            .scaleEffect(scale, anchor: .topLeading)
            .frame(
                width: max(1, contentSize.width * scale),
                height: max(1, contentSize.height * scale),
                alignment: .topLeading
            )
            .animation(.easeInOut(duration: 0.16), value: scale)
    }
}

private struct TeamGridInspector: View {
    let agent: NativeAgentSummary?
    let modelLabel: String
    let tasks: [TeamOperationsTask]
    let sources: [TeamGridInformationSource]
    let files: [TeamGridFileAccess]
    let initiative: TeamGridInitiativeResponse?
    let permissions: TeamGridPermissionConfig?
    let spawnSubagent: () -> Void
    let addTeamAgent: () -> Void
    @Environment(\.colorScheme) private var colorScheme

    private var agentSources: [TeamGridInformationSource] {
        guard let agent else { return [] }
        return sources.filter { $0.agents.contains(agent.id) }
    }

    private var agentFiles: [TeamGridFileAccess] {
        guard let agent else { return [] }
        return Array(files.filter { $0.agents.contains(agent.id) }.prefix(5))
    }

    var body: some View {
        if let agent {
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    VStack(alignment: .leading, spacing: 5) {
                        Text("SELECTED AGENT")
                            .font(.system(size: 9, weight: .semibold, design: .rounded))
                            .tracking(1.2).foregroundStyle(JameBrand.orange)
                        Text(agent.name.isEmpty ? agent.id : agent.name)
                            .font(.system(size: 23, weight: .semibold, design: .rounded))
                        Text(agent.id).font(.caption.monospaced()).foregroundStyle(teamMuted(colorScheme))
                    }

                    HStack(spacing: 18) {
                        TeamGridMetric(value: agent.sessionCount ?? 0, label: "SESSIONS")
                        TeamGridMetric(value: agent.messageCount ?? 0, label: "MESSAGES")
                        TeamGridMetric(value: agent.toolCalls ?? 0, label: "TOOLS")
                    }

                    inspectorSection("Role") {
                        Text(agent.human?.persona?.isEmpty == false ? agent.human!.persona! : "No persona configured")
                            .font(.subheadline).foregroundStyle(teamPrimary(colorScheme))
                    }
                    inspectorSection("Manages") {
                        Text(agent.human?.memoryNotes?.isEmpty == false ? agent.human!.memoryNotes! : "No management responsibility saved")
                            .font(.subheadline).foregroundStyle(teamPrimary(colorScheme))
                    }
                    inspectorSection("Runtime") {
                        Label(modelLabel, systemImage: "cpu")
                            .foregroundStyle(JameBrand.orange)
                        Label(agent.workspace.isEmpty ? "Default workspace" : agent.workspace, systemImage: "folder")
                            .lineLimit(2).truncationMode(.middle)
                    }
                    inspectorSection("Modification access") {
                        if let permissions {
                            let writeTools = [
                                permissions.tools.writeFile?.enabled == true ? "create" : nil,
                                permissions.tools.editFile?.enabled == true ? "edit" : nil,
                                permissions.tools.appendFile?.enabled == true ? "append" : nil,
                            ].compactMap { $0 }
                            Label(
                                writeTools.isEmpty ? "File modification disabled" : "Can \(writeTools.joined(separator: ", ")) files",
                                systemImage: writeTools.isEmpty ? "lock.fill" : "pencil.and.outline"
                            )
                            if permissions.agents.defaults.restrictToWorkspace {
                                Label(agent.workspace.isEmpty ? permissions.agents.defaults.workspace : agent.workspace, systemImage: "folder.badge.checkmark")
                                    .lineLimit(3).truncationMode(.middle)
                                if !permissions.tools.allowWritePaths.isEmpty {
                                    Text("Plus \(permissions.tools.allowWritePaths.count) approved external write area\(permissions.tools.allowWritePaths.count == 1 ? "" : "s").")
                                }
                            } else {
                                Label("Workspace restriction is off", systemImage: "exclamationmark.triangle.fill")
                                    .foregroundStyle(JameBrand.orange)
                            }
                        } else {
                            Text("Modification permissions unavailable.")
                        }
                    }
                    inspectorSection("Delegated tasks") {
                        if tasks.isEmpty {
                            Text("No task delegated yet.")
                                .foregroundStyle(teamMuted(colorScheme))
                        } else {
                            ForEach(tasks.prefix(5)) { task in
                                Label(task.title, systemImage: task.status == "done" ? "checkmark.circle" : "flag.fill")
                                    .lineLimit(2)
                                Text(task.status.replacingOccurrences(of: "_", with: " ").uppercased())
                                    .font(.caption2.monospaced())
                                    .foregroundStyle(JameBrand.orange)
                            }
                        }
                    }
                    inspectorSection("Skills & delegation") {
                        Text((agent.skills ?? []).isEmpty ? "No dedicated skills" : (agent.skills ?? []).joined(separator: ", "))
                        Text((agent.subagents ?? []).isEmpty ? "No spawned agents" : "Delegates to: \((agent.subagents ?? []).joined(separator: ", "))")
                    }
                    if agent.id == "main", let initiative {
                        inspectorSection("Initiative") {
                            Label(
                                initiative.initiative ? "Proactive discovery enabled" : "Task-list checks only",
                                systemImage: initiative.initiative ? "sparkles" : "pause.circle"
                            )
                            Text("Checks every \(initiative.intervalMinutes) minutes")
                            if let latest = initiative.latest {
                                Text(latest.summary)
                                    .lineLimit(5)
                                    .foregroundStyle(teamPrimary(colorScheme))
                            } else {
                                Text("Waiting for the first proactive check.")
                            }
                        }
                    }
                    inspectorSection("Information movement") {
                        if agentSources.isEmpty && agentFiles.isEmpty {
                            Text("No recorded source or file movement yet.")
                                .foregroundStyle(teamMuted(colorScheme))
                        } else {
                            ForEach(agentSources) { source in
                                Label("\(source.name) · \(source.messages) messages", systemImage: "arrow.down.right")
                            }
                            ForEach(agentFiles) { file in
                                Label(URL(fileURLWithPath: file.path).lastPathComponent, systemImage: "doc")
                                    .help(file.path)
                            }
                        }
                    }

                    VStack(spacing: 9) {
                        Button("Spawn under \(agent.name.isEmpty ? agent.id : agent.name)", action: spawnSubagent)
                            .buttonStyle(.borderedProminent).tint(JameBrand.orange)
                            .frame(maxWidth: .infinity)
                        Button("Add independent team agent", action: addTeamAgent)
                            .buttonStyle(.bordered)
                            .frame(maxWidth: .infinity)
                    }
                }
                .padding(22)
            }
        } else {
            ContentUnavailableView("Select an agent", systemImage: "person.crop.circle.badge.questionmark")
        }
    }

    @ViewBuilder
    private func inspectorSection<Content: View>(_ title: String, @ViewBuilder content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title.uppercased())
                .font(.system(size: 9, weight: .semibold, design: .rounded))
                .tracking(1).foregroundStyle(JameBrand.orange)
            VStack(alignment: .leading, spacing: 7) { content() }
                .font(.caption).foregroundStyle(teamMuted(colorScheme))
        }
        .padding(.leading, 12)
        .overlay(alignment: .leading) { Rectangle().fill(JameBrand.orange.opacity(0.7)).frame(width: 2) }
    }
}

private struct TeamGridMetric: View {
    let value: Int
    let label: String
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text("\(value)").font(.title3.weight(.semibold)).foregroundStyle(teamPrimary(colorScheme))
            Text(label).font(.system(size: 8, weight: .semibold, design: .rounded)).foregroundStyle(teamMuted(colorScheme))
        }
    }
}

private struct TeamGridInformationSources: View {
    let sources: [TeamGridInformationSource]
    @Environment(\.colorScheme) private var colorScheme

    private func icon(for source: TeamGridInformationSource) -> String {
        switch source.id {
        case "jame": return "bubble.left.and.bubble.right.fill"
        case "terminal": return "terminal.fill"
        case "telegram": return "paperplane.fill"
        case "discord": return "gamecontroller.fill"
        case "slack": return "number"
        default: return "arrow.down.circle.fill"
        }
    }

    var body: some View {
        VStack(spacing: 12) {
            Label("Information flows into JameClaw", systemImage: "arrow.down.to.line.compact")
                .font(.headline)
                .foregroundStyle(JameBrand.orange)
            Text("Only sources with saved incoming messages are shown.")
                .font(.caption)
                .foregroundStyle(.secondary)
            HStack(alignment: .top, spacing: 14) {
                ForEach(sources) { source in
                    VStack(alignment: .leading, spacing: 7) {
                        HStack(spacing: 6) {
                            Image(systemName: icon(for: source)).foregroundStyle(JameBrand.orange)
                            Text("INPUT").font(.system(size: 9, weight: .semibold, design: .rounded))
                                .foregroundStyle(JameBrand.orange)
                            Spacer()
                        }
                        Text(source.name).font(.headline).lineLimit(1)
                        Text("\(source.messages) incoming message\(source.messages == 1 ? "" : "s") · \(source.sessions) session\(source.sessions == 1 ? "" : "s")")
                            .font(.caption).foregroundStyle(.secondary)
                        Text(source.agents.joined(separator: ", "))
                            .font(.caption2.monospaced()).foregroundStyle(.secondary).lineLimit(1)
                    }
                    .padding(13)
                    .frame(width: 180, alignment: .leading)
                    .background(teamPanel(colorScheme), in: Rectangle())
                    .overlay(Rectangle().stroke(JameBrand.orange.opacity(0.35), lineWidth: 1))
                }
            }
            Text("Jame Chat combines the currently shared Desktop and Web Console history.")
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: 620)
    }
}

private struct TeamGridMCPSection: View {
    let servers: [MCPServer]
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        VStack(spacing: 12) {
            TeamGridFlowLine(length: 22)
            Label("MCP servers connected to JameClaw", systemImage: "cable.connector")
                .font(.headline)
                .foregroundStyle(JameBrand.orange)
            if servers.isEmpty {
                Text("No MCP servers are configured yet.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 170), spacing: 14)], spacing: 14) {
                    ForEach(servers) { server in
                        VStack(alignment: .leading, spacing: 6) {
                            HStack(spacing: 6) {
                                Image(systemName: server.enabled ? "cable.connector" : "cable.connector.slash")
                                Text(server.enabled ? "CONNECTED" : "DISABLED")
                                    .font(.system(size: 9, weight: .semibold, design: .rounded))
                                Spacer()
                            }
                            .foregroundStyle(server.enabled ? JameBrand.orange : teamMuted(colorScheme))
                            Text(server.name).font(.headline).lineLimit(1)
                            Text(server.transport.uppercased())
                                .font(.caption.weight(.semibold)).foregroundStyle(.secondary)
                            Text(server.endpoint).font(.caption.monospaced()).foregroundStyle(.secondary).lineLimit(1)
                        }
                        .padding(13)
                        .frame(width: 190, alignment: .leading)
                        .background(teamPanel(colorScheme), in: Rectangle())
                        .overlay(Rectangle().stroke(JameBrand.orange.opacity(server.enabled ? 0.40 : 0.18), lineWidth: 1))
                    }
                }
            }
        }
        .frame(maxWidth: 760)
    }
}

private struct TeamGridModificationSection: View {
    let permissions: TeamGridPermissionConfig
    let artifacts: [TeamGridArtifact]
    @Environment(\.colorScheme) private var colorScheme

    private var enabledWriteTools: [String] {
        [
            permissions.tools.writeFile?.enabled == true ? "CREATE" : nil,
            permissions.tools.editFile?.enabled == true ? "EDIT" : nil,
            permissions.tools.appendFile?.enabled == true ? "APPEND" : nil,
        ].compactMap { $0 }
    }

    private var modificationAreas: [String] {
        guard permissions.agents.defaults.restrictToWorkspace else {
            return ["Host filesystem — workspace restriction is off"]
        }
        var seen = Set<String>()
        return ([permissions.agents.defaults.workspace] + permissions.tools.allowWritePaths)
            .map { NSString(string: $0).expandingTildeInPath }
            .filter { !$0.isEmpty && seen.insert($0).inserted }
    }

    private var artifactsPath: String {
        let workspace = NSString(string: permissions.agents.defaults.workspace).expandingTildeInPath
        return URL(fileURLWithPath: workspace, isDirectory: true)
            .appendingPathComponent("artifacts", isDirectory: true).path
    }

    var body: some View {
        VStack(spacing: 12) {
            TeamGridFlowLine(length: 22)
            VStack(spacing: 3) {
                Label("What JameClaw can modify", systemImage: "pencil.and.outline")
                    .font(.headline)
                    .foregroundStyle(JameBrand.orange)
                Text("Live workspace restrictions and approved write areas from JameClaw settings.")
                    .font(.caption)
                    .foregroundStyle(teamMuted(colorScheme))
            }

            HStack(alignment: .top, spacing: 14) {
                VStack(alignment: .leading, spacing: 12) {
                    HStack(spacing: 7) {
                        Image(systemName: enabledWriteTools.isEmpty ? "lock.fill" : "folder.badge.gearshape")
                        Text("MODIFICATION ACCESS")
                            .font(.system(size: 10, weight: .semibold, design: .rounded))
                            .tracking(0.8)
                        Spacer()
                        Text(enabledWriteTools.isEmpty ? "READ ONLY" : "ENABLED")
                            .font(.system(size: 9, weight: .bold, design: .rounded))
                    }
                    .foregroundStyle(JameBrand.orange)

                    if enabledWriteTools.isEmpty {
                        Text("Create, edit, and append tools are disabled.")
                            .font(.caption)
                            .foregroundStyle(teamMuted(colorScheme))
                    } else {
                        Text(enabledWriteTools.joined(separator: "  ·  "))
                            .font(.caption2.monospaced().weight(.semibold))
                            .foregroundStyle(teamPrimary(colorScheme))
                    }

                    ForEach(Array(modificationAreas.enumerated()), id: \.offset) { index, path in
                        VStack(alignment: .leading, spacing: 4) {
                            Label(index == 0 ? "PRIMARY WORKSPACE" : "APPROVED WRITE AREA", systemImage: index == 0 ? "internaldrive" : "checkmark.shield.fill")
                                .font(.system(size: 9, weight: .semibold, design: .rounded))
                                .foregroundStyle(JameBrand.orange)
                            Text(path)
                                .font(.caption.monospaced())
                                .foregroundStyle(teamPrimary(colorScheme))
                                .lineLimit(3)
                                .truncationMode(.middle)
                                .textSelection(.enabled)
                        }
                        .padding(10)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(teamSurface(colorScheme), in: Rectangle())
                    }
                }
                .padding(15)
                .frame(width: 390, alignment: .leading)
                .background(teamPanel(colorScheme), in: Rectangle())
                .overlay(Rectangle().stroke(JameBrand.orange.opacity(0.45)))

                VStack(alignment: .leading, spacing: 12) {
                    HStack(spacing: 7) {
                        Image(systemName: "shippingbox.fill")
                        Text("ARTIFACTS")
                            .font(.system(size: 10, weight: .semibold, design: .rounded))
                            .tracking(0.8)
                        Spacer()
                        Text("\(artifacts.count)")
                            .font(.caption.monospaced().weight(.bold))
                            .padding(.horizontal, 7).padding(.vertical, 3)
                            .background(JameBrand.orange, in: Rectangle())
                            .foregroundStyle(JameBrand.ink)
                    }
                    .foregroundStyle(JameBrand.orange)

                    Text("Apps, websites, and code projects JameClaw can create and edit.")
                        .font(.caption)
                        .foregroundStyle(teamMuted(colorScheme))
                    Text(artifactsPath)
                        .font(.caption2.monospaced())
                        .foregroundStyle(teamMuted(colorScheme))
                        .lineLimit(2)
                        .truncationMode(.middle)
                        .textSelection(.enabled)

                    if artifacts.isEmpty {
                        VStack(alignment: .leading, spacing: 5) {
                            Label("No artifacts yet", systemImage: "plus.square.dashed")
                                .font(.subheadline.weight(.semibold))
                            Text("Ask JameClaw to build an app, website, or code artifact and it will appear in this box.")
                                .font(.caption)
                                .foregroundStyle(teamMuted(colorScheme))
                        }
                        .padding(12)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(teamSurface(colorScheme), in: Rectangle())
                    } else {
                        ForEach(artifacts.prefix(5)) { artifact in
                            HStack(spacing: 9) {
                                Image(systemName: artifact.kind == "app" ? "macwindow" : "chevron.left.forwardslash.chevron.right")
                                    .foregroundStyle(JameBrand.orange)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(artifact.name).font(.subheadline.weight(.semibold)).lineLimit(1)
                                    Text("\(artifact.kind.uppercased()) · \(Date(timeIntervalSince1970: TimeInterval(artifact.updatedAtMS) / 1000).formatted(date: .abbreviated, time: .shortened))")
                                        .font(.caption2).foregroundStyle(teamMuted(colorScheme))
                                }
                            }
                        }
                    }
                }
                .padding(15)
                .frame(width: 390, alignment: .leading)
                .background(teamPanel(colorScheme), in: Rectangle())
                .overlay(Rectangle().stroke(JameBrand.orange.opacity(0.45)))
            }
        }
        .frame(maxWidth: 820)
    }
}

private struct TeamGridFileSection: View {
    let files: [TeamGridFileAccess]

    var body: some View {
        VStack(spacing: 12) {
            TeamGridFlowLine(length: 22)
            VStack(spacing: 3) {
                Label("Files accessed by JameClaw", systemImage: "folder.badge.gearshape")
                    .font(.headline)
                    .foregroundStyle(JameBrand.orange)
                Text("Each box grows with the number of recorded read or write operations.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            if files.isEmpty {
                Text("No recorded file activity yet. This map uses saved JameClaw tool calls, not macOS-wide file monitoring.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: 440)
            } else {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 130), spacing: 14)], spacing: 14) {
                    ForEach(Array(files.prefix(30))) { file in
                        TeamGridFileNode(file: file)
                    }
                }
            }
        }
        .frame(maxWidth: 820)
    }
}

private struct TeamGridFileNode: View {
    let file: TeamGridFileAccess
    @Environment(\.colorScheme) private var colorScheme

    private var size: CGFloat {
        CGFloat(min(210, max(128, 118 + file.accesses * 12)))
    }

    private var displayName: String {
        URL(fileURLWithPath: file.path).lastPathComponent.isEmpty ? file.path : URL(fileURLWithPath: file.path).lastPathComponent
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Image(systemName: "doc.fill").foregroundStyle(JameBrand.orange)
            Text(displayName).font(.subheadline.weight(.semibold)).lineLimit(2)
            Text("\(file.accesses) access\(file.accesses == 1 ? "" : "es")")
                .font(.caption.weight(.medium)).foregroundStyle(JameBrand.orange)
            if !file.agents.isEmpty {
                Text(file.agents.joined(separator: ", "))
                    .font(.caption.monospaced()).foregroundStyle(.secondary).lineLimit(1)
            }
        }
        .padding(13)
        .frame(width: size, height: max(105, size * 0.72), alignment: .leading)
        .background(teamPanel(colorScheme), in: Rectangle())
        .overlay(Rectangle().stroke(JameBrand.orange.opacity(0.38), lineWidth: 1))
        .help(file.path)
    }
}

private struct AgentTreeBranch: View {
    let agent: NativeAgentSummary
    let agentsByID: [String: NativeAgentSummary]
    let visited: Set<String>
    let selectedAgentID: String
    let modelLabel: (NativeAgentSummary) -> String
    let activeTask: (NativeAgentSummary) -> TeamOperationsTask?
    let delegatableTasks: [TeamOperationsTask]
    let delegateTask: (TeamOperationsTask, NativeAgentSummary) -> Void
    let createTask: (NativeAgentSummary) -> Void
    let createLoop: (NativeAgentSummary) -> Void
    let select: (String) -> Void

    private var children: [NativeAgentSummary] {
        guard !visited.contains(agent.id) else { return [] }
        return (agent.subagents ?? []).compactMap { agentsByID[$0] }
    }

    var body: some View {
        VStack(spacing: 0) {
            TeamGridNode(
                agent: agent,
                kind: agent.id == "main" ? .main : .spawned,
                isSelected: selectedAgentID == agent.id,
                modelLabel: modelLabel(agent),
                activeTask: activeTask(agent),
                delegatableTasks: delegatableTasks,
                delegateTask: { delegateTask($0, agent) },
                createTask: { createTask(agent) },
                createLoop: { createLoop(agent) },
                select: { select(agent.id) }
            )
            if !children.isEmpty {
                TeamGridFlowLine(length: 28)
                HStack(alignment: .top, spacing: 18) {
                    ForEach(children) { child in
                        VStack(spacing: 0) {
                            TeamGridFlowLine(length: 20)
                            AgentTreeBranch(
                                agent: child,
                                agentsByID: agentsByID,
                                visited: visited.union([agent.id]),
                                selectedAgentID: selectedAgentID,
                                modelLabel: modelLabel,
                                activeTask: activeTask,
                                delegatableTasks: delegatableTasks,
                                delegateTask: delegateTask,
                                createTask: createTask,
                                createLoop: createLoop,
                                select: select
                            )
                        }
                    }
                }
            }
        }
    }
}

private struct TeamGridNode: View {
    enum Kind { case main, spawned, team }
    let agent: NativeAgentSummary
    let kind: Kind
    let isSelected: Bool
    let modelLabel: String
    let activeTask: TeamOperationsTask?
    let delegatableTasks: [TeamOperationsTask]
    let delegateTask: (TeamOperationsTask) -> Void
    let createTask: () -> Void
    let createLoop: () -> Void
    let select: () -> Void
    @AppStorage("launcher.design.teamGlow") private var teamGlow = true
    @Environment(\.colorScheme) private var colorScheme
    private var label: String {
        switch kind {
        case .main: return "JAMECLAW"
        case .spawned: return "SPAWNED"
        case .team: return "TEAM"
        }
    }

    private var identitySymbol: String {
        agentIdentitySymbol(agent)
    }

    var body: some View {
        Button(action: select) {
            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 6) {
                    Image(systemName: identitySymbol)
                        .accessibilityLabel(kind == .main ? "JameClaw icon" : "Unique \(agent.name.isEmpty ? agent.id : agent.name) icon")
                    Text(label).font(.system(size: 9, weight: .semibold, design: .rounded))
                    Spacer()
                    if (agent.messageCount ?? 0) > 0 {
                        TeamGridLivePulse()
                    }
                }
                .foregroundStyle(isSelected ? JameBrand.ink : JameBrand.orange)
                Text(agent.name.isEmpty ? agent.id : agent.name)
                    .font(.headline)
                    .foregroundStyle(isSelected ? JameBrand.ink : teamPrimary(colorScheme))
                    .lineLimit(1)
                if agent.human?.persona?.isEmpty == false {
                    Text(agent.human!.persona!)
                        .font(.caption)
                        .foregroundStyle(isSelected ? JameBrand.ink.opacity(0.64) : teamMuted(colorScheme))
                        .lineLimit(2)
                }
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    Image(systemName: "cpu")
                        .font(.system(size: 10, weight: .semibold))
                    Text(modelLabel)
                        .font(.system(size: 10, weight: .semibold, design: .monospaced))
                        .lineLimit(2)
                }
                .foregroundStyle(isSelected ? JameBrand.ink.opacity(0.78) : JameBrand.orange)
                .padding(.horizontal, 7)
                .padding(.vertical, 5)
                .background(
                    isSelected ? JameBrand.ink.opacity(0.08) : JameBrand.orange.opacity(0.10),
                    in: Rectangle()
                )
                VStack(alignment: .leading, spacing: 4) {
                    HStack(spacing: 5) {
                        Image(systemName: activeTask == nil ? "tray" : activeTask?.nodeKind == "loop" ? "repeat" : "flag.fill")
                        Text(activeTask == nil ? "NO ACTIVE TASK" : (activeTask?.nodeKind == "loop" ? "LOOP · " : "") + activeTask!.status.uppercased())
                    }
                    .font(.system(size: 9, weight: .semibold, design: .rounded))
                    Text(activeTask?.title ?? "Right-click to delegate work")
                        .font(.caption2.weight(activeTask == nil ? .regular : .semibold))
                        .lineLimit(2)
                }
                .foregroundStyle(isSelected ? JameBrand.ink.opacity(0.72) : teamMuted(colorScheme))
                HStack(spacing: 8) {
                    Label("\(agent.sessionCount ?? 0)", systemImage: "bubble.left")
                    Label("\(agent.toolCalls ?? 0)", systemImage: "wrench.and.screwdriver")
                }
                .font(.caption2.weight(.semibold))
                .foregroundStyle(isSelected ? JameBrand.ink.opacity(0.64) : teamMuted(colorScheme))
            }
            .padding(14)
            .frame(width: 230, alignment: .leading)
            .background(isSelected ? JameBrand.orange : teamPanel(colorScheme), in: Rectangle())
            .overlay(Rectangle().stroke(JameBrand.orange.opacity(isSelected ? 1 : 0.42), lineWidth: isSelected ? 2 : 1))
            .shadow(
                color: teamGlow ? JameBrand.orange.opacity(isSelected ? 0.62 : 0.34) : .clear,
                radius: teamGlow ? (isSelected ? 16 : 10) : 0
            )
        }
        .buttonStyle(.plain)
        .contextMenu {
            Button("Select agent") { select() }
            Divider()
            if delegatableTasks.isEmpty {
                Button("No planned tasks available") { }
                    .disabled(true)
            } else {
                Menu("Delegate existing task") {
                    ForEach(delegatableTasks) { task in
                        Button(task.title) { delegateTask(task) }
                    }
                }
            }
            Button("Create task for \(agent.name.isEmpty ? agent.id : agent.name)…") {
                createTask()
            }
            Button("Create loop for \(agent.name.isEmpty ? agent.id : agent.name)…", systemImage: "repeat") {
                createLoop()
            }
        }
    }
}

private struct TeamGridLivePulse: View {
    @State private var pulsing = false

    var body: some View {
        Circle()
            .fill(pulsing ? JameBrand.orange : JameBrand.orange.opacity(0.35))
            .frame(width: 7, height: 7)
            .scaleEffect(pulsing ? 1.15 : 0.72)
            .animation(.easeInOut(duration: 0.8).repeatForever(autoreverses: true), value: pulsing)
            .onAppear { pulsing = true }
            .accessibilityLabel("Recorded activity")
    }
}

private func AgentField(_ label: String, _ value: String) -> some View {
    Group {
        Text(label).foregroundStyle(.secondary)
        Text(value).textSelection(.enabled)
    }
}

private func AgentMetric(_ label: String, _ value: Int) -> some View {
    VStack(alignment: .leading, spacing: 3) {
        Text(label).font(.caption).foregroundStyle(.secondary)
        Text("\(value)").font(.title3.weight(.semibold))
    }
}

private enum AgentCreationMode {
    case team
    case subagent
}

private struct CreateAgentView: View {
    let mode: AgentCreationMode
    let parent: NativeAgentSummary?
    @ObservedObject var store: NativeAgentStore
    let created: (String) -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var id = ""
    @State private var name = ""
    @State private var workspace = ""
    @State private var role = ""
    @State private var task = ""
    @State private var company = ""
    @State private var manages = ""

    private var cleanManagement: String {
        manages.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var managementMemory: String {
        "Manages: \(cleanManagement)\nUse this as the agent's durable ownership boundary when accepting or delegating work."
    }

    private var teamContext: String {
        var lines = ["Role: \(role.trimmingCharacters(in: .whitespacesAndNewlines))"]
        lines.append("Manages: \(cleanManagement)")
        let cleanTask = task.trimmingCharacters(in: .whitespacesAndNewlines)
        let cleanCompany = company.trimmingCharacters(in: .whitespacesAndNewlines)
        if !cleanTask.isEmpty { lines.append("Task: \(cleanTask)") }
        if !cleanCompany.isEmpty { lines.append("Company: \(cleanCompany)") }
        return lines.joined(separator: "\n")
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
            Text(mode == .team ? "Add team agent" : "Spawn subagent").font(.title2.weight(.semibold))
            Text(mode == .team
                ? "Create an independent full agent that appears in the JameClaw team directory. It is not nested as a subagent."
                : "Create an agent managed by \(parent?.name.isEmpty == false ? parent!.name : parent?.id ?? "main"). It can be delegated work by its parent.")
                .foregroundStyle(.secondary)
            if mode == .team {
                HStack {
                    Text("Detected on this Mac").font(.headline)
                    Spacer()
                    Button { store.discoverLocalAgents() } label: { Image(systemName: "arrow.clockwise") }
                        .help("Scan installed agent tools again")
                }
                if store.localAgents.isEmpty {
                    Text("No supported agent tools were found. Install a CLI or desktop app, then refresh this list.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(store.localAgents) { localAgent in
                        Button {
                            apply(localAgent.preset)
                        } label: {
                            HStack {
                                Image(systemName: "checkmark.circle.fill").foregroundStyle(.green)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(localAgent.title).fontWeight(.semibold)
                                    Text(localAgent.location).font(.caption.monospaced()).foregroundStyle(.secondary).lineLimit(1)
                                }
                                Spacer()
                                Text("Use").foregroundStyle(.tint)
                            }
                            .contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)
                        .padding(8)
                        .background(Color.secondary.opacity(0.08))
                        .clipShape(Rectangle())
                    }
                }
            }
            if mode == .team {
                VStack(alignment: .leading, spacing: 9) {
                    Text("Team agent templates").font(.headline)
                    Text("Choose a starting profile, then refine its workspace and role below.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 132), spacing: 9)], spacing: 9) {
                        ForEach(TeamAgentPreset.allCases) { preset in
                            Button { apply(preset) } label: {
                                HStack(spacing: 8) {
                                    Image(systemName: preset.symbol)
                                        .foregroundStyle(preset.tint)
                                        .frame(width: 18)
                                    VStack(alignment: .leading, spacing: 1) {
                                        Text(preset.title).font(.subheadline.weight(.semibold))
                                        Text(preset.summary).font(.caption2).foregroundStyle(.secondary).lineLimit(1)
                                    }
                                    Spacer(minLength: 0)
                                }
                                .padding(9)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(Color.secondary.opacity(0.08), in: Rectangle())
                            }
                            .buttonStyle(.plain)
                            .help("Use the \(preset.title) team-agent profile")
                        }
                    }
                }
            }
            TextField("Agent ID (for example, researcher)", text: $id)
            TextField("Display name", text: $name)
            TextField("Workspace (optional)", text: $workspace)
            VStack(alignment: .leading, spacing: 5) {
                Text("What does this agent manage? *")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(JameBrand.orange)
                TextField("Required — for example, releases, product research, or customer support", text: $manages, axis: .vertical)
                    .lineLimit(2...4)
                Text("Saved in this agent's memory and shown in Team Grid.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            if mode == .team {
                VStack(alignment: .leading, spacing: 5) {
                    Text("Role *").font(.caption.weight(.semibold)).foregroundStyle(JameBrand.orange)
                    TextField("Required — for example, Lead researcher", text: $role, axis: .vertical)
                        .lineLimit(2...4)
                }
                TextField("Task (optional)", text: $task, axis: .vertical)
                    .lineLimit(2...4)
                TextField("Company (optional)", text: $company)
            } else {
                TextField("Role or persona (optional)", text: $role, axis: .vertical)
                    .lineLimit(2...4)
            }
            HStack {
                Spacer()
                Button("Cancel") { dismiss() }
                Button(store.isCreating ? "Creating…" : (mode == .team ? "Create team agent" : "Spawn subagent")) {
                    Task {
                        let didCreate = await store.createAgent(
                            id: id,
                            name: name,
                            workspace: workspace,
                            persona: mode == .team ? teamContext : role,
                            memoryNotes: managementMemory,
                            parentID: mode == .subagent ? (parent?.id ?? "main") : nil,
                            managedByMain: mode == .subagent
                        )
                        if didCreate { created(id.trimmingCharacters(in: .whitespacesAndNewlines)) }
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(
                    store.isCreating
                        || id.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                        || cleanManagement.isEmpty
                        || (mode == .team && role.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                )
            }
            }
            .padding(24)
        }
        .frame(width: 520, height: 640)
    }

    private func apply(_ preset: TeamAgentPreset) {
        id = preset.id
        name = preset.title
        role = preset.persona
    }
}

private enum TeamAgentPreset: String, CaseIterable, Identifiable {
    case codex, kimi, claudeCode = "claude-code", hermes, grok, nanobot, openCode = "open-code", aider, gemini, goose, qwenCode = "qwen-code", cursor

    var id: String { rawValue }
    var title: String {
        switch self {
        case .codex: return "Codex"
        case .kimi: return "Kimi"
        case .claudeCode: return "Claude Code"
        case .hermes: return "Hermes"
        case .grok: return "Grok"
        case .nanobot: return "Nanobot"
        case .openCode: return "OpenCode"
        case .aider: return "Aider"
        case .gemini: return "Gemini CLI"
        case .goose: return "Goose"
        case .qwenCode: return "Qwen Code"
        case .cursor: return "Cursor"
        }
    }
    var summary: String {
        switch self {
        case .codex: return "Implementation"
        case .kimi: return "Research"
        case .claudeCode: return "Code review"
        case .hermes: return "Autonomous work"
        case .grok: return "Fast product thinking"
        case .nanobot: return "Lightweight helper"
        case .openCode: return "Open-source coding"
        case .aider: return "Repository editing"
        case .gemini: return "Multimodal research"
        case .goose: return "Tool-assisted work"
        case .qwenCode: return "Coding assistance"
        case .cursor: return "IDE collaboration"
        }
    }
    var symbol: String {
        switch self {
        case .codex: return "chevron.left.forwardslash.chevron.right"
        case .kimi: return "text.book.closed"
        case .claudeCode: return "checkmark.seal"
        case .hermes: return "bolt.fill"
        case .grok: return "sparkle.magnifyingglass"
        case .nanobot: return "cpu"
        case .openCode: return "terminal"
        case .aider: return "hammer"
        case .gemini: return "diamond"
        case .goose: return "wrench.and.screwdriver"
        case .qwenCode: return "curlybraces"
        case .cursor: return "cursorarrow.rays"
        }
    }
    var tint: Color {
        switch self {
        case .codex, .openCode: return .green
        case .kimi, .grok: return .purple
        case .claudeCode: return .orange
        case .hermes: return .blue
        case .nanobot: return .cyan
        case .aider: return .pink
        case .gemini: return .blue
        case .goose: return .orange
        case .qwenCode: return .purple
        case .cursor: return .indigo
        }
    }
    var persona: String {
        switch self {
        case .codex: return "Implementation-focused coding teammate."
        case .kimi: return "Research and long-context analysis teammate."
        case .claudeCode: return "Code review and software engineering teammate."
        case .hermes: return "Independent planning and execution teammate."
        case .grok: return "Fast, candid product and technical reasoning teammate."
        case .nanobot: return "Lightweight, focused helper for concise tasks and reliable follow-through."
        case .openCode: return "Open-source coding teammate for implementation and code exploration."
        case .aider: return "Repository-aware editing teammate focused on precise code changes."
        case .gemini: return "Multimodal research and analysis teammate for broad technical questions."
        case .goose: return "Tool-assisted teammate for structured tasks and repeatable workflows."
        case .qwenCode: return "Practical coding teammate for implementation, debugging, and code explanation."
        case .cursor: return "IDE-oriented teammate for collaborative software development and code navigation."
        }
    }
}

private struct NativeModelInfo: Codable, Identifiable {
    let modelName: String
    let model: String
    let apiBase: String?
    let configured: Bool

    var id: String { modelName }

    enum CodingKeys: String, CodingKey {
        case modelName = "model_name"
        case model
        case apiBase = "api_base"
        case configured
    }
}

private struct NativeModelsResponse: Codable {
    let models: [NativeModelInfo]
    let defaultModel: String
    let modelFallbacks: [String]?

    enum CodingKeys: String, CodingKey {
        case models
        case defaultModel = "default_model"
        case modelFallbacks = "model_fallbacks"
    }
}

private struct NativeProviderInfo: Codable, Identifiable {
    let id: String
    let name: String
    let description: String
    let requiresAPIKey: Bool
    let keyLabel: String?
    let recommendedModels: [NativeProviderModelPreset]
    let configuredModels: [String]?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case description
        case requiresAPIKey = "requires_api_key"
        case keyLabel = "key_label"
        case recommendedModels = "recommended_models"
        case configuredModels = "configured_models"
    }
}

private struct NativeProviderModelPreset: Codable, Identifiable {
    let id: String
    let name: String
    let requiresAPIKey: Bool
    let keyLabel: String?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case requiresAPIKey = "requires_api_key"
        case keyLabel = "key_label"
    }
}

private struct NativeProviderCatalogResponse: Codable {
    let providers: [NativeProviderInfo]
}

private struct NativeDiscoveredProviderModel: Codable, Identifiable {
    let id: String
    let name: String
    let ownedBy: String?

    enum CodingKeys: String, CodingKey {
        case id, name
        case ownedBy = "owned_by"
    }
}

private struct NativeProviderModelDiscoveryResponse: Codable {
    let models: [NativeDiscoveredProviderModel]
}

@MainActor
private final class NativeProviderStore: ObservableObject {
    @Published var models: [NativeModelInfo] = []
    @Published var defaultModel = ""
    @Published var selectedModel = ""
    @Published var selectedFallbackModel = ""
    @Published var providerNames: [String: String] = [:]
    @Published var catalog: [NativeProviderInfo] = []
    @Published var status = ""
    @Published var isLoading = false

    func load(port: Int) async {
        isLoading = true
        defer { isLoading = false }

        do {
            // The configured-model list is the source of truth for the native
            // selector. The catalog is only used for friendly provider names;
            // a missing catalog must never hide a working CLI/local provider.
            let modelsResponse: NativeModelsResponse = try await fetch(path: "/api/models", port: port)
            models = modelsResponse.models.filter(\.configured)
            defaultModel = modelsResponse.defaultModel
            selectedModel = modelsResponse.defaultModel
            selectedFallbackModel = modelsResponse.modelFallbacks?.first ?? ""
            if let catalogResponse: NativeProviderCatalogResponse = try? await fetch(path: "/api/models/catalog", port: port) {
                catalog = catalogResponse.providers
                providerNames = Dictionary(uniqueKeysWithValues: catalogResponse.providers.flatMap { provider in
                    (provider.configuredModels ?? []).map { ($0, provider.name) }
                })
            } else {
                catalog = []
                providerNames = [:]
            }
            status = models.isEmpty
                ? "No configured AI providers yet."
                : ""
        } catch {
            status = "Could not load AI providers. Start JameClaw and try again."
        }
    }

    func setDefaultModel(_ modelName: String, port: Int) async {
        guard modelName != defaultModel else { return }
        do {
            var request = URLRequest(url: authenticatedConsoleURL(port: port, path: "/api/models/default"))
            request.httpMethod = "POST"
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONEncoder().encode(["model_name": modelName, "role": "chat"])
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            let restart = authenticatedConsoleRequest(port: port, path: "/api/gateway/restart", method: "POST")
            let (_, restartResponse) = try await URLSession.shared.data(for: restart)
            guard let http = restartResponse as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            defaultModel = modelName
            selectedModel = modelName
            status = "Primary AI provider changed. Gateway restarted."
        } catch {
            status = "Could not change the primary AI provider."
        }
    }

    func setFailover(primaryModel: String, fallbackModel: String, port: Int) async {
        guard !primaryModel.isEmpty else { return }
        guard !fallbackModel.isEmpty else {
            await setDefaultModel(primaryModel, port: port)
            status = "Chat model updated without a fallback provider."
            return
        }
        guard primaryModel != fallbackModel else {
            status = "Choose a different fallback provider."
            return
        }
        do {
            var request = authenticatedConsoleRequest(port: port, path: "/api/models/failover", method: "POST")
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: [
                "primary_model": primaryModel,
                "secondary_model": fallbackModel,
            ])
            let (data, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            let result = (try? JSONSerialization.jsonObject(with: data) as? [String: Any]) ?? [:]
            defaultModel = primaryModel
            selectedModel = primaryModel
            selectedFallbackModel = fallbackModel
            let restarted = result["gateway_restarted"] as? Bool ?? false
            status = restarted
                ? "Failover saved. JameClaw will try the fallback provider automatically."
                : "Failover saved. Restart the gateway to apply it."
        } catch {
            status = "Could not save the fallback provider."
        }
    }

    func addCatalogModel(
        provider: NativeProviderInfo,
        preset: NativeProviderModelPreset?,
        remoteModelID: String?,
        apiKey: String,
        setAsPrimary: Bool,
        port: Int
    ) async -> Bool {
        do {
            var request = authenticatedConsoleRequest(port: port, path: "/api/models/from-catalog", method: "POST")
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: [
                "provider_id": provider.id,
                "preset_id": preset?.id ?? "",
                "remote_model_id": remoteModelID?.trimmingCharacters(in: .whitespacesAndNewlines) ?? "",
                "api_key": apiKey.trimmingCharacters(in: .whitespacesAndNewlines),
                "set_default": setAsPrimary,
            ])
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            if setAsPrimary {
                let restart = authenticatedConsoleRequest(port: port, path: "/api/gateway/restart", method: "POST")
                let (_, restartResponse) = try await URLSession.shared.data(for: restart)
                guard let http = restartResponse as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                    throw URLError(.badServerResponse)
                }
            }
            await load(port: port)
            if setAsPrimary {
                status = "New AI provider connected and set as primary."
            } else {
                selectedFallbackModel = models.first(where: { $0.modelName != selectedModel && providerNames[$0.modelName] == provider.name })?.modelName ?? selectedFallbackModel
                status = "Provider added. Choose it as the global fallback."
            }
            return true
        } catch {
            status = "Could not add this provider. Check the model and API key."
            return false
        }
    }

    func discoverCatalogModels(provider: NativeProviderInfo, apiKey: String, port: Int) async -> [NativeDiscoveredProviderModel] {
        do {
            var request = authenticatedConsoleRequest(port: port, path: "/api/models/catalog/discover", method: "POST")
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: [
                "provider_id": provider.id,
                "api_key": apiKey.trimmingCharacters(in: .whitespacesAndNewlines),
            ])
            let (data, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            return (try JSONDecoder().decode(NativeProviderModelDiscoveryResponse.self, from: data)).models
        } catch {
            status = "Could not fetch models from this API. Check the API key and provider access."
            return []
        }
    }

    func providerName(for model: NativeModelInfo) -> String {
        if let configuredProvider = providerNames[model.modelName] {
            return configuredProvider
        }
        guard let apiBase = model.apiBase,
              let host = URL(string: apiBase)?.host,
              !host.isEmpty else { return "Custom provider" }
        return host
    }

    private func fetch<T: Decodable>(path: String, port: Int) async throws -> T {
        let (data, response) = try await URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: path))
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw URLError(.badServerResponse)
        }
        return try JSONDecoder().decode(T.self, from: data)
    }
}

private struct NativeResearchProviderInfo: Codable, Identifiable {
    let id: String
    let name: String
    let description: String
    let requiresAPIKey: Bool
    let configured: Bool
    let enabled: Bool

    enum CodingKeys: String, CodingKey {
        case id, name, description, configured, enabled
        case requiresAPIKey = "requires_api_key"
    }
}

private struct NativeResearchProviderResponse: Codable {
    let providers: [NativeResearchProviderInfo]
    let activeProvider: String

    enum CodingKeys: String, CodingKey {
        case providers
        case activeProvider = "active_provider"
    }
}

@MainActor
private final class NativeResearchProviderStore: ObservableObject {
    @Published var providers: [NativeResearchProviderInfo] = []
    @Published var selectedProviderID = ""
    @Published var activeProviderID = ""
    @Published var status = ""
    @Published var isLoading = false

    func load(port: Int, preserveStatus: Bool = false) async {
        isLoading = true
        if !preserveStatus { status = "" }
        defer { isLoading = false }
        do {
            let (data, response) = try await URLSession.shared.data(
                from: authenticatedConsoleURL(port: port, path: "/api/research-providers")
            )
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            let result = try JSONDecoder().decode(NativeResearchProviderResponse.self, from: data)
            providers = result.providers
            activeProviderID = result.activeProvider
            if selectedProviderID.isEmpty {
                selectedProviderID = result.activeProvider.isEmpty ? (result.providers.first?.id ?? "") : result.activeProvider
            }
        } catch {
            status = "Could not load research providers."
        }
    }

    func connect(apiKey: String, port: Int) async {
        guard !selectedProviderID.isEmpty else {
            status = "Choose a research provider."
            return
        }
        await save(providerID: selectedProviderID, apiKey: apiKey, enabled: true, port: port)
    }

    func disconnect(port: Int) async {
        guard !activeProviderID.isEmpty else { return }
        await save(providerID: activeProviderID, apiKey: "", enabled: false, port: port)
    }

    private func save(providerID: String, apiKey: String, enabled: Bool, port: Int) async {
        isLoading = true
        status = enabled ? "Connecting research provider…" : "Disconnecting research provider…"
        defer { isLoading = false }
        do {
            var request = authenticatedConsoleRequest(port: port, path: "/api/research-providers/\(providerID)", method: "POST")
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: [
                "api_key": apiKey.trimmingCharacters(in: .whitespacesAndNewlines),
                "enabled": enabled,
            ])
            let (data, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                let detail = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)
                throw NSError(
                    domain: "JameClawResearch",
                    code: (response as? HTTPURLResponse)?.statusCode ?? 1,
                    userInfo: [NSLocalizedDescriptionKey: detail?.isEmpty == false ? detail! : "Could not save the research provider."]
                )
            }

            var restarted = false
            let restart = authenticatedConsoleRequest(port: port, path: "/api/gateway/restart", method: "POST")
            if let restartResponse = try? await URLSession.shared.data(for: restart).1 as? HTTPURLResponse {
                restarted = (200..<300).contains(restartResponse.statusCode)
            }
            await load(port: port, preserveStatus: true)
            if enabled {
                status = restarted
                    ? "Research provider connected and ready."
                    : "Research provider connected. Restart the gateway to apply it."
            } else {
                status = restarted
                    ? "Research provider disconnected."
                    : "Research provider disconnected. Restart the gateway to apply it."
            }
        } catch {
            status = error.localizedDescription.isEmpty ? "Could not connect the research provider." : error.localizedDescription
        }
    }
}

private enum AgentPersonalityPreset: String, CaseIterable, Identifiable {
    case helpful
    case concise
    case creative
    case analytical
    case custom

    var id: String { rawValue }

    var label: String {
        switch self {
        case .helpful: return "Helpful guide"
        case .concise: return "Concise operator"
        case .creative: return "Creative collaborator"
        case .analytical: return "Analytical thinker"
        case .custom: return "Custom"
        }
    }

    var instructions: String {
        switch self {
        case .helpful:
            return "Be warm, practical, and proactive. Explain decisions clearly, anticipate useful next steps, and help the user make steady progress."
        case .concise:
            return "Be direct and efficient. Lead with the answer, keep explanations compact, and avoid unnecessary sections or repetition."
        case .creative:
            return "Be imaginative and collaborative. Offer original options, make thoughtful connections, and keep ideas grounded in something the user can act on."
        case .analytical:
            return "Be precise and evidence-driven. State assumptions, separate facts from inferences, identify tradeoffs, and check important details before concluding."
        case .custom:
            return ""
        }
    }
}

struct QuickSettingsView: View {
    @ObservedObject var settings: LauncherSettingsStore
    let openArchivedChats: () -> Void
    @StateObject private var providers = NativeProviderStore()
    @StateObject private var researchProviders = NativeResearchProviderStore()
    @AppStorage("launcher.design.theme") private var savedTheme = LauncherTheme.light.rawValue
    @AppStorage("launcher.design.accent") private var savedAccent = LauncherAccent.theme.rawValue
    @AppStorage("launcher.design.density") private var savedDensity = ChatDensity.comfortable.rawValue
    @AppStorage("launcher.design.surface") private var savedSurface = MessageSurface.cards.rawValue
    @AppStorage("launcher.design.fontScale") private var fontScale = 1.0
    @AppStorage("launcher.design.backgroundPath") private var backgroundPath = ""
    @AppStorage("launcher.design.windowOpacity") private var windowOpacity = 1.0
    @AppStorage("launcher.design.teamGlow") private var teamGlow = true
    @AppStorage("launcher.safety.documentApprovalPolicy") private var documentApprovalPolicy = DocumentApprovalPolicy.outsideWorkspace.rawValue
    @AppStorage("jame.notifications.taskCompletion") private var taskCompletionNotifications = true
    @State private var showingBackgroundPicker = false
    @State private var showingProviderSetup = false
    @State private var providerSetupPurpose = ProviderSetupPurpose.primary
    @State private var allowOpenMacApps = false
    @State private var allowMusicPlaylists = false
    @State private var musicPlaylistStatus = ""
    @State private var documentSafetyStatus = ""
    @State private var researchAPIKey = ""
    @State private var personalityPreset: AgentPersonalityPreset = .custom
    @State private var personalityInstructions = ""
    @State private var personalityAgentName = ""
    @State private var personalityTone = ""
    @State private var personalityDiscussionMode = ""
    @State private var personalityMemoryNotes = ""
    @State private var personalityStatusStyle = ""
    @State private var personalityStatus = ""
    @State private var agentUpdateStatus = ""
    @State private var agentCurrentVersion = ""
    @State private var agentLatestVersion = ""
    @State private var agentUpdateAvailable = false
    @State private var agentUpdateLoading = false

    private var selectedResearchProvider: NativeResearchProviderInfo? {
        researchProviders.providers.first(where: { $0.id == researchProviders.selectedProviderID })
    }

    private var activeDesignPreset: NativeDesignPreset {
        NativeDesignPreset.allCases.first(where: { preset in
            guard let configuration = preset.configuration else { return false }
            return savedTheme == configuration.theme.rawValue
                && savedAccent == configuration.accent.rawValue
                && savedDensity == configuration.density.rawValue
                && savedSurface == configuration.surface.rawValue
                && abs(fontScale - configuration.fontScale) < 0.001
                && abs(windowOpacity - configuration.windowOpacity) < 0.001
                && teamGlow == configuration.teamGlow
                && backgroundPath.isEmpty
        }) ?? .custom
    }

    private var designPresetSelection: Binding<String> {
        Binding(
            get: { activeDesignPreset.rawValue },
            set: { newValue in
                guard let preset = NativeDesignPreset(rawValue: newValue) else { return }
                applyDesignPreset(preset)
            }
        )
    }

    var body: some View {
        Form {
            Section("Conversation history") {
                Button(action: openArchivedChats) {
                    Label("Archived chats", systemImage: "archivebox.fill")
                }
                Text("Open chats removed from the active Sessions timeline. You can restore any chat from there.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Section("Notifications") {
                Toggle("Notify me when Jame finishes a task", isOn: $taskCompletionNotifications)
                    .onChange(of: taskCompletionNotifications) { _, enabled in
                        guard enabled else { return }
                        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound]) { _, _ in }
                    }
                Text("When JameClaw is in the background, send a macOS notification with a short result preview when an agent turn completes.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Section("JameClaw AI agent") {
                HStack {
                    VStack(alignment: .leading, spacing: 3) {
                        Text("Desktop agent version")
                            .font(.subheadline.weight(.medium))
                        Text(agentCurrentVersion.isEmpty ? "Version not checked yet" : agentCurrentVersion)
                            .font(.caption.monospaced())
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    if agentUpdateAvailable {
                        Label(agentLatestVersion.isEmpty ? "Update available" : agentLatestVersion, systemImage: "arrow.down.circle.fill")
                            .font(.caption.weight(.medium))
                            .foregroundStyle(.orange)
                    }
                }
                HStack {
                    Button(agentUpdateLoading ? "Checking…" : "Check for updates") {
                        Task { await checkForAgentUpdate() }
                    }
                    .disabled(agentUpdateLoading)
                    if agentUpdateAvailable {
                        Button("Open update page") {
                            Task { await openAgentUpdate() }
                        }
                        .buttonStyle(.borderedProminent)
                    }
                }
                if !agentUpdateStatus.isEmpty {
                    Text(agentUpdateStatus)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Text("Check for a newer JameClaw AI agent release and open the official update page when one is available.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Section("Safety") {
                Picker("Document access approvals", selection: $documentApprovalPolicy) {
                    ForEach(DocumentApprovalPolicy.allCases) { policy in
                        Text(policy.label).tag(policy.rawValue)
                    }
                }
                if let policy = DocumentApprovalPolicy(rawValue: documentApprovalPolicy) {
                    Text(policy.detail)
                        .font(.caption)
                        .foregroundStyle(policy == .yolo ? Color.red : Color.secondary)
                }
                Label(
                    documentApprovalPolicy == DocumentApprovalPolicy.yolo.rawValue
                        ? "YOLO removes the workspace restriction for file tools."
                        : "File tools stay restricted to Jame's active workspace.",
                    systemImage: documentApprovalPolicy == DocumentApprovalPolicy.yolo.rawValue
                        ? "exclamationmark.triangle.fill"
                        : "lock.shield.fill"
                )
                    .font(.caption)
                    .foregroundStyle(documentApprovalPolicy == DocumentApprovalPolicy.yolo.rawValue ? Color.red : JameBrand.orange)
                HStack {
                    Button("Save document safety") { saveDocumentSafety() }
                        .buttonStyle(.borderedProminent)
                    if !documentSafetyStatus.isEmpty {
                        Text(documentSafetyStatus).font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
            Section("Web Console") {
                TextField("Port", text: $settings.port)
                Toggle("Allow devices on my local network", isOn: $settings.lanAccess)
            }
            if settings.lanAccess {
                Section("Allowed network CIDRs") {
                    TextEditor(text: $settings.allowedCIDRs)
                        .font(.system(.body, design: .monospaced))
                        .frame(height: 65)
                    Text("One CIDR per line, for example 192.168.1.0/24.")
                        .font(.caption).foregroundStyle(.secondary)
                }
            }
            Section("Personality") {
                Picker("Personality style", selection: $personalityPreset) {
                    ForEach(AgentPersonalityPreset.allCases) { preset in
                        Text(preset.label).tag(preset)
                    }
                }
                .onChange(of: personalityPreset) { _, preset in
                    if preset != .custom {
                        personalityInstructions = preset.instructions
                    }
                }
                if personalityPreset == .custom {
                    TextEditor(text: $personalityInstructions)
                        .font(.system(.body, design: .rounded))
                        .frame(minHeight: 105)
                    Text("Write custom instructions for how the main agent should speak and make decisions.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    Text("Jame will automatically use the built-in \(personalityPreset.label) personality.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                HStack {
                    Button("Save personality") {
                        savePersonality()
                    }
                    .buttonStyle(.borderedProminent)
                    if !personalityStatus.isEmpty {
                        Text(personalityStatus).font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
            Section {
                Button("Save settings") { settings.save() }
                if !settings.saveStatus.isEmpty { Text(settings.saveStatus).font(.caption).foregroundStyle(.secondary) }
            }
            Section("AI Provider") {
                if providers.models.isEmpty {
                    Text(providers.isLoading ? "Loading configured providers…" : "No configured AI providers.")
                        .foregroundStyle(.secondary)
                } else {
                    Picker("Primary provider", selection: $providers.selectedModel) {
                        ForEach(providers.models) { model in
                            Text("\(providers.providerName(for: model)) · \(model.modelName)")
                                .tag(model.modelName)
                        }
                    }
                    Picker("Fallback provider", selection: $providers.selectedFallbackModel) {
                        Text("No fallback").tag("")
                        ForEach(providers.models.filter { $0.modelName != providers.selectedModel }) { model in
                            Text("\(providers.providerName(for: model)) · \(model.modelName)")
                                .tag(model.modelName)
                        }
                    }
                    Text("If the primary provider has a retriable failure, JameClaw will automatically try the fallback provider.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    if providers.selectedFallbackModel.isEmpty {
                        HStack(spacing: 10) {
                            Button("Add fallback provider") {
                                openProviderSetup(.fallback)
                            }
                            .buttonStyle(.bordered)
                            Text("Add another provider or model, then select it here as the fallback.")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }
                    HStack {
                        Button("Save provider failover") {
                            Task {
                                await providers.setFailover(
                                    primaryModel: providers.selectedModel,
                                    fallbackModel: providers.selectedFallbackModel,
                                    port: Int(settings.port) ?? 18800
                                )
                            }
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(providers.selectedModel.isEmpty)
                        Text("Current primary: \(providers.models.first(where: { $0.modelName == providers.defaultModel }).map { providers.providerName(for: $0) } ?? "Not selected")")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    HStack {
                        Button("Change primary provider") {
                            Task {
                                await providers.setDefaultModel(
                                    providers.selectedModel,
                                    port: Int(settings.port) ?? 18800
                                )
                            }
                        }
                        .buttonStyle(.bordered)
                        .disabled(providers.selectedModel.isEmpty || providers.selectedModel == providers.defaultModel)
                        Button("Add new AI provider") {
                            openProviderSetup(.primary)
                        }
                        .buttonStyle(.borderedProminent)
                    }
                }
                if providers.models.isEmpty {
                    Button("Add new AI provider") {
                        openProviderSetup(.primary)
                    }
                    .buttonStyle(.borderedProminent)
                }
                HStack {
                    Button("Refresh providers") {
                        Task { await providers.load(port: Int(settings.port) ?? 18800) }
                    }
                    if providers.isLoading { ProgressView().controlSize(.small) }
                }
                if !providers.status.isEmpty {
                    Text(providers.status).font(.caption).foregroundStyle(.secondary)
                }
            }
            Section("Research Provider") {
                if researchProviders.providers.isEmpty {
                    Text(researchProviders.isLoading ? "Loading research providers…" : "No research providers available.")
                        .foregroundStyle(.secondary)
                } else {
                    Picker("Dedicated research provider", selection: $researchProviders.selectedProviderID) {
                        ForEach(researchProviders.providers) { provider in
                            Text(provider.name).tag(provider.id)
                        }
                    }
                    if let selectedResearchProvider {
                        Text(selectedResearchProvider.description)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        Label(
                            selectedResearchProvider.enabled
                                ? "Connected and active"
                                : selectedResearchProvider.configured ? "Key saved — connect when ready" : "Not connected",
                            systemImage: selectedResearchProvider.enabled ? "checkmark.circle.fill" : "magnifyingglass.circle"
                        )
                        .font(.caption.weight(.medium))
                        .foregroundStyle(selectedResearchProvider.enabled ? Color.green : Color.secondary)
                        if selectedResearchProvider.requiresAPIKey {
                            SecureField(
                                selectedResearchProvider.configured
                                    ? "API key saved — leave blank to reuse it"
                                    : "Research provider API key",
                                text: $researchAPIKey
                            )
                            Text("The API key is stored in JameClaw's private security configuration and is never returned to the interface.")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        } else {
                            Text("No API key is required for this provider.")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        HStack {
                            Button(selectedResearchProvider.enabled ? "Update connection" : "Connect research provider") {
                                Task {
                                    await researchProviders.connect(
                                        apiKey: researchAPIKey,
                                        port: Int(settings.port) ?? 18800
                                    )
                                    researchAPIKey = ""
                                }
                            }
                            .buttonStyle(.borderedProminent)
                            .disabled(
                                researchProviders.isLoading
                                    || (selectedResearchProvider.requiresAPIKey
                                        && !selectedResearchProvider.configured
                                        && researchAPIKey.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                            )
                            if !researchProviders.activeProviderID.isEmpty {
                                Button("Disconnect") {
                                    Task { await researchProviders.disconnect(port: Int(settings.port) ?? 18800) }
                                }
                                .buttonStyle(.bordered)
                                .disabled(researchProviders.isLoading)
                            }
                            if researchProviders.isLoading { ProgressView().controlSize(.small) }
                        }
                    }
                }
                Text("Research providers power JameClaw's source-backed web research without changing the primary AI chat provider.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                HStack {
                    Button("Refresh research providers") {
                        Task { await researchProviders.load(port: Int(settings.port) ?? 18800) }
                    }
                    if !researchProviders.status.isEmpty {
                        Text(researchProviders.status).font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
            Section("Desktop app commands") {
                Toggle("Allow opening Mac apps", isOn: $allowOpenMacApps)
                Toggle("Allow Apple Music playlists", isOn: $allowMusicPlaylists)
                Text("Type @ in Chat to select any installed Mac app. Apple Music can also create named playlists.")
                    .font(.caption).foregroundStyle(.secondary)
                HStack {
                    Button("Save app command permissions") { saveMusicPlaylistPermission() }
                    if !musicPlaylistStatus.isEmpty {
                        Text(musicPlaylistStatus).font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
            Section("Design") {
                Picker("Full design theme", selection: designPresetSelection) {
                    ForEach(NativeDesignPreset.allCases) { preset in
                        Text(preset.label).tag(preset.rawValue)
                    }
                }
                Text("One selection applies theme, accent, chat spacing, message style, text size, transparency, team glow, and the default background.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                Label(activeDesignPreset.detail, systemImage: activeDesignPreset == .custom ? "slider.horizontal.3" : "paintpalette.fill")
                    .font(.caption.weight(.medium))
                    .foregroundStyle(.tint)
                Divider()
                Picker("Theme", selection: $savedTheme) {
                    ForEach(LauncherTheme.allCases) { theme in
                        Text(theme.label).tag(theme.rawValue)
                    }
                }
                Picker("Accent color", selection: $savedAccent) {
                    ForEach(LauncherAccent.allCases) { accent in
                        Text(accent.label).tag(accent.rawValue)
                    }
                }
                Picker("Chat spacing", selection: $savedDensity) {
                    ForEach(ChatDensity.allCases) { density in
                        Text(density.label).tag(density.rawValue)
                    }
                }
                Picker("Message style", selection: $savedSurface) {
                    ForEach(MessageSurface.allCases) { surface in
                        Text(surface.label).tag(surface.rawValue)
                    }
                }
                Picker("Chat text size", selection: $fontScale) {
                    Text("Small").tag(0.88)
                    Text("Default").tag(1.0)
                    Text("Large").tag(1.15)
                    Text("Extra large").tag(1.3)
                }
                VStack(alignment: .leading, spacing: 6) {
                    HStack {
                        Text("Window transparency")
                        Spacer()
                        Text("\(Int(windowOpacity * 100))%")
                            .font(.caption.monospaced())
                            .foregroundStyle(.secondary)
                    }
                    Slider(value: $windowOpacity, in: 0.65...1.0, step: 0.05)
                    Text("Lower the value to see more of the desktop behind JameClaw.")
                        .font(.caption).foregroundStyle(.secondary)
                }
                Toggle("Glow team grid", isOn: $teamGlow)
                Text("Adds an orange live glow around team and subagent cards in the Team Grid.")
                    .font(.caption).foregroundStyle(.secondary)
                HStack {
                    Button("Choose chat background") { showingBackgroundPicker = true }
                    if !backgroundPath.isEmpty {
                        Button("Use default background") { backgroundPath = "" }
                    }
                }
                Text("Chat uses the Creation of Adam artwork across the full canvas by default. Choose an image if you want a custom local background.")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Section {
                Text("Developed by Jame")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .formStyle(.grouped)
        .padding(.top, 6)
        .task {
            let normalizedTheme = launcherThemePreference(from: savedTheme).rawValue
            if savedTheme != normalizedTheme { savedTheme = normalizedTheme }
            let port = Int(settings.port) ?? 18800
            await providers.load(port: port)
            await researchProviders.load(port: port)
            await loadMusicPlaylistPermission(port: port)
            await loadPersonality(port: port)
        }
        .fileImporter(
            isPresented: $showingBackgroundPicker,
            allowedContentTypes: [.image],
            allowsMultipleSelection: false
        ) { result in
            guard case let .success(urls) = result, let sourceURL = urls.first else { return }
            saveChatBackground(from: sourceURL)
        }
        .sheet(isPresented: $showingProviderSetup, onDismiss: {
            Task { await providers.load(port: Int(settings.port) ?? 18800) }
        }) {
            ProviderSetupSheet(
                port: Int(settings.port) ?? 18800,
                providers: providers,
                purpose: providerSetupPurpose
            ) {
                showingProviderSetup = false
            }
        }
    }

    private func openProviderSetup(_ purpose: ProviderSetupPurpose) {
        providerSetupPurpose = purpose
        showingProviderSetup = true
    }

    private func checkForAgentUpdate() async {
        agentUpdateLoading = true
        agentUpdateStatus = ""
        defer { agentUpdateLoading = false }
        do {
            let (data, response) = try await URLSession.shared.data(
                from: authenticatedConsoleURL(port: Int(settings.port) ?? 18800, path: "/api/update/status")
            )
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            let status = try JSONSerialization.jsonObject(with: data) as? [String: Any] ?? [:]
            agentCurrentVersion = status["current_version"] as? String ?? ""
            agentLatestVersion = status["latest_version"] as? String ?? ""
            agentUpdateAvailable = status["update_available"] as? Bool ?? false
            if let error = status["check_error"] as? String, !error.isEmpty {
                agentUpdateStatus = "Could not check for updates: \(error)"
            } else if agentUpdateAvailable {
                agentUpdateStatus = "A newer JameClaw AI agent release is available."
            } else {
                agentUpdateStatus = "JameClaw is up to date."
            }
        } catch {
            agentUpdateStatus = "Could not check for JameClaw updates."
        }
    }

    private func openAgentUpdate() async {
        do {
            var request = authenticatedConsoleRequest(port: Int(settings.port) ?? 18800, path: "/api/update/open", method: "POST")
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            let (_, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            agentUpdateStatus = "Opened the official JameClaw update page."
        } catch {
            agentUpdateStatus = "Could not open the JameClaw update page."
        }
    }

    private func loadPersonality(port: Int) async {
        do {
            let (data, response) = try await URLSession.shared.data(
                from: authenticatedConsoleURL(port: port, path: "/api/agents")
            )
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            let main = try JSONDecoder().decode(NativeAgentsResponse.self, from: data)
                .agents.first(where: { $0.id == "main" })
            let human = main?.human
            personalityAgentName = human?.agentName ?? ""
            personalityInstructions = human?.persona ?? ""
            personalityTone = human?.tone ?? ""
            personalityDiscussionMode = human?.discussionMode ?? ""
            personalityMemoryNotes = human?.memoryNotes ?? ""
            personalityStatusStyle = human?.statusStyle ?? ""
            personalityPreset = AgentPersonalityPreset.allCases.first {
                $0 != .custom && $0.instructions == personalityInstructions
            } ?? .custom
        } catch {
            personalityStatus = "Could not load the agent personality."
        }
    }

    private func savePersonality() {
        personalityStatus = "Saving…"
        Task {
            do {
                let port = Int(settings.port) ?? 18800
                var request = authenticatedConsoleRequest(port: port, path: "/api/agents/main", method: "PATCH")
                request.setValue("application/json", forHTTPHeaderField: "Content-Type")
                request.httpBody = try JSONEncoder().encode(
                    NativeUpdateAgentRequest(
                        human: NativeUpdateAgentHuman(
                            agentName: personalityAgentName,
                            persona: personalityInstructions,
                            tone: personalityTone,
                            discussionMode: personalityDiscussionMode,
                            memoryNotes: personalityMemoryNotes,
                            statusStyle: personalityStatusStyle
                        )
                    )
                )
                let (_, response) = try await URLSession.shared.data(for: request)
                guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                    throw URLError(.badServerResponse)
                }
                personalityStatus = "Personality saved."
            } catch {
                personalityStatus = "Could not save the agent personality."
            }
        }
    }

    private func applyDesignPreset(_ preset: NativeDesignPreset) {
        guard let configuration = preset.configuration else { return }
        withAnimation(.easeInOut(duration: 0.18)) {
            savedTheme = configuration.theme.rawValue
            savedAccent = configuration.accent.rawValue
            savedDensity = configuration.density.rawValue
            savedSurface = configuration.surface.rawValue
            fontScale = configuration.fontScale
            windowOpacity = configuration.windowOpacity
            teamGlow = configuration.teamGlow
            backgroundPath = ""
        }
    }

    private func saveDocumentSafety() {
        documentSafetyStatus = "Saving…"
        Task {
            do {
                let port = Int(settings.port) ?? 18800
                var request = authenticatedConsoleRequest(port: port, path: "/api/config", method: "PATCH")
                request.setValue("application/json", forHTTPHeaderField: "Content-Type")
                request.httpBody = try JSONSerialization.data(withJSONObject: [
                    "agents": ["defaults": [
                        "restrict_to_workspace": documentApprovalPolicy != DocumentApprovalPolicy.yolo.rawValue,
                    ]],
                ])
                let (_, response) = try await URLSession.shared.data(for: request)
                guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                    throw URLError(.badServerResponse)
                }
                let restart = authenticatedConsoleRequest(port: port, path: "/api/gateway/restart", method: "POST")
                let (_, restartResponse) = try await URLSession.shared.data(for: restart)
                guard let http = restartResponse as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                    throw URLError(.badServerResponse)
                }
                documentSafetyStatus = documentApprovalPolicy == DocumentApprovalPolicy.yolo.rawValue
                    ? "YOLO saved. Document prompts and workspace restriction are off."
                    : "Saved. Workspace restriction is active."
            } catch {
                documentSafetyStatus = "Could not save safety settings."
            }
        }
    }

    private func saveChatBackground(from sourceURL: URL) {
        let didAccess = sourceURL.startAccessingSecurityScopedResource()
        defer { if didAccess { sourceURL.stopAccessingSecurityScopedResource() } }
        let destinationDirectory = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".jameclaw")
        let fileExtension = sourceURL.pathExtension.isEmpty ? "jpg" : sourceURL.pathExtension
        let destinationURL = destinationDirectory.appendingPathComponent("chat-background.\(fileExtension)")
        do {
            try FileManager.default.createDirectory(at: destinationDirectory, withIntermediateDirectories: true)
            if FileManager.default.fileExists(atPath: destinationURL.path) {
                try FileManager.default.removeItem(at: destinationURL)
            }
            try FileManager.default.copyItem(at: sourceURL, to: destinationURL)
            backgroundPath = destinationURL.path
        } catch {
            settings.saveStatus = "Could not save the selected background image."
        }
    }

    private func loadMusicPlaylistPermission(port: Int) async {
        do {
            let (data, response) = try await URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: "/api/config"))
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode),
                  let root = try JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let tools = root["tools"] as? [String: Any],
                  let macControl = tools["mac_control"] as? [String: Any] else {
                throw URLError(.badServerResponse)
            }
            allowOpenMacApps = macControl["allow_open_apps"] as? Bool ?? false
            allowMusicPlaylists = macControl["allow_music_playlists"] as? Bool ?? false
        } catch {
            musicPlaylistStatus = "Could not load app command permissions."
        }
    }

    private func saveMusicPlaylistPermission() {
        Task {
            do {
                var request = URLRequest(url: authenticatedConsoleURL(port: Int(settings.port) ?? 18800, path: "/api/config"))
                request.httpMethod = "PATCH"
                request.setValue("application/json", forHTTPHeaderField: "Content-Type")
                request.httpBody = try JSONSerialization.data(withJSONObject: [
                    "tools": ["mac_control": [
                        "allow_open_apps": allowOpenMacApps,
                        "allow_music_playlists": allowMusicPlaylists,
                    ]],
                ])
                let (_, response) = try await URLSession.shared.data(for: request)
                guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                    throw URLError(.badServerResponse)
                }
                let restart = authenticatedConsoleRequest(port: Int(settings.port) ?? 18800, path: "/api/gateway/restart", method: "POST")
                let (_, restartResponse) = try await URLSession.shared.data(for: restart)
                guard let http = restartResponse as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                    throw URLError(.badServerResponse)
                }
                musicPlaylistStatus = "App command permissions saved. Gateway restarted."
            } catch {
                musicPlaylistStatus = "Could not save app command permissions."
            }
        }
    }
}

struct NativeChatMessage: Identifiable {
    let id: String
    let role: String
    var content: String
}

private enum NativePlanStepStatus: String {
    case pending
    case inProgress = "in_progress"
    case completed
    case cancelled
}

private struct NativePlanStep: Identifiable {
    let id: String
    var title: String
    var tools: [String]
    var status: NativePlanStepStatus
}

private func nativePlanSteps(from content: String) -> [NativePlanStep]? {
    let lowered = content.lowercased()
    guard lowered.contains("**plan**") || lowered.hasPrefix("🧭 plan") else { return nil }

    var steps: [NativePlanStep] = []
    for rawLine in content.split(whereSeparator: \.isNewline) {
        var line = rawLine.trimmingCharacters(in: .whitespacesAndNewlines)
        guard line.hasPrefix("-") || line.hasPrefix("•") else { continue }
        line = String(line.dropFirst()).trimmingCharacters(in: .whitespaces)

        let status: NativePlanStepStatus
        if line.lowercased().hasPrefix("[x]") {
            status = .completed
            line = String(line.dropFirst(3)).trimmingCharacters(in: .whitespaces)
        } else if line.hasPrefix("[>]") {
            status = .inProgress
            line = String(line.dropFirst(3)).trimmingCharacters(in: .whitespaces)
        } else if line.hasPrefix("[-]") {
            status = .cancelled
            line = String(line.dropFirst(3)).trimmingCharacters(in: .whitespaces)
        } else if line.hasPrefix("[ ]") {
            status = .pending
            line = String(line.dropFirst(3)).trimmingCharacters(in: .whitespaces)
        } else {
            status = .pending
        }

        var title = line
        var toolNames: [String] = []
        if let toolRange = line.range(of: "Tools:", options: .caseInsensitive) {
            title = String(line[..<toolRange.lowerBound])
                .trimmingCharacters(in: CharacterSet(charactersIn: " \t—–-:("))
            let toolText = String(line[toolRange.upperBound...])
                .trimmingCharacters(in: CharacterSet(charactersIn: " \t.)"))
            toolNames = toolText
                .split(separator: ",")
                .map { $0.trimmingCharacters(in: CharacterSet(charactersIn: " `\t")) }
                .filter { !$0.isEmpty && $0.lowercased() != "none" }
        }
        guard !title.isEmpty else { continue }
        steps.append(NativePlanStep(
            id: "plan-step-\(steps.count)",
            title: title,
            tools: toolNames,
            status: status
        ))
    }
    return steps.isEmpty ? nil : steps
}

private func nativePlanContent(from steps: [NativePlanStep]) -> String {
    let lines = steps.map { step in
        let marker: String
        switch step.status {
        case .pending: marker = "[ ]"
        case .inProgress: marker = "[>]"
        case .completed: marker = "[x]"
        case .cancelled: marker = "[-]"
        }
        let tools = step.tools.isEmpty ? "" : " — Tools: " + step.tools.joined(separator: ", ")
        return "- \(marker) \(step.title)\(tools)"
    }
    return "🧭 **Plan**\n\n" + lines.joined(separator: "\n")
}

private struct NativePlanCard: View {
    let steps: [NativePlanStep]
    let accent: Color
    let fontScale: CGFloat

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 8) {
                Image(systemName: "checklist")
                    .foregroundStyle(accent)
                Text("EXECUTION PLAN")
                    .font(.system(size: 11 * fontScale, weight: .semibold, design: .monospaced))
                    .foregroundStyle(accent)
                Spacer()
                Text("\(steps.filter { $0.status == .completed }.count)/\(steps.count)")
                    .font(.caption.monospacedDigit())
                    .foregroundStyle(.secondary)
            }

            ForEach(steps) { step in
                HStack(alignment: .top, spacing: 10) {
                    Image(systemName: symbol(for: step.status))
                        .foregroundStyle(color(for: step.status))
                        .font(.system(size: 16 * fontScale, weight: .semibold))
                        .frame(width: 18)
                    VStack(alignment: .leading, spacing: 6) {
                        Text(step.title)
                            .font(.system(size: 13 * fontScale, weight: step.status == .inProgress ? .semibold : .regular))
                            .foregroundStyle(step.status == .cancelled ? .secondary : .primary)
                            .strikethrough(step.status == .completed || step.status == .cancelled)
                        if !step.tools.isEmpty {
                            HStack(spacing: 5) {
                                Image(systemName: "wrench.and.screwdriver.fill")
                                Text(step.tools.joined(separator: "  ·  "))
                                    .lineLimit(2)
                            }
                            .font(.system(size: 10 * fontScale, weight: .medium, design: .monospaced))
                            .foregroundStyle(.secondary)
                            .padding(.horizontal, 7)
                            .padding(.vertical, 4)
                            .background(Color.white.opacity(0.06), in: Rectangle())
                        }
                    }
                    Spacer(minLength: 0)
                }
            }
        }
        .padding(14)
        .background(JameBrand.elevated.opacity(0.96), in: Rectangle())
        .overlay(Rectangle().stroke(accent.opacity(0.42)))
    }

    private func symbol(for status: NativePlanStepStatus) -> String {
        switch status {
        case .pending: return "square"
        case .inProgress: return "square.dotted"
        case .completed: return "checkmark.square.fill"
        case .cancelled: return "xmark.square.fill"
        }
    }

    private func color(for status: NativePlanStepStatus) -> Color {
        switch status {
        case .pending: return .secondary
        case .inProgress, .completed: return accent
        case .cancelled: return .secondary
        }
    }
}

private struct NativeMemoryChangedCard: View {
    let summary: String
    let accent: Color
    let fontScale: CGFloat

    var body: some View {
        HStack(alignment: .top, spacing: 11) {
            Image(systemName: "brain.head.profile.fill")
                .font(.system(size: 17 * fontScale, weight: .semibold))
                .foregroundStyle(accent)
                .frame(width: 22)
            VStack(alignment: .leading, spacing: 5) {
                Text("MEMORY WAS UPDATED")
                    .font(.system(size: 11 * fontScale, weight: .semibold, design: .monospaced))
                    .foregroundStyle(accent)
                Text(summary)
                    .font(.system(size: 12 * fontScale, design: .monospaced))
                    .foregroundStyle(.primary)
                    .textSelection(.enabled)
            }
            Spacer(minLength: 0)
            Image(systemName: "checkmark.square.fill")
                .foregroundStyle(accent)
        }
        .padding(12)
        .background(accent.opacity(0.09), in: Rectangle())
        .overlay(alignment: .leading) {
            Rectangle()
                .fill(accent)
                .frame(width: 3)
        }
        .overlay(Rectangle().stroke(accent.opacity(0.42)))
    }
}

private struct PendingNativeChatMessage {
    let id: String
    let content: String
    let modelOverride: String
}

struct NativeAppError: Identifiable {
    let id = UUID()
    let title: String
    let detail: String
}

private enum SkillUploadError: LocalizedError {
    case chooseSkillFile
    case missingSkillFile
    case alreadyExists(String)

    var errorDescription: String? {
        switch self {
        case .chooseSkillFile:
            return "Choose a file named SKILL.md or a folder containing one."
        case .missingSkillFile:
            return "This folder does not contain SKILL.md."
        case let .alreadyExists(name):
            return "A workspace skill named \(name) already exists."
        }
    }
}

@MainActor
final class NativeChatStore: ObservableObject {
	@Published var messages: [NativeChatMessage] = []
	@Published var activity: [String] = []
    @Published var draft = ""
    @Published var agentName = "Jame"
    @Published var status = "Connecting…"
    @Published var isThinking = false
    @Published var isExecutingPlan = false
    @Published var queuedMessageCount = 0
    @Published var lastError: NativeAppError?
    @Published var workspaceName = "Choose workspace"
    @Published var workspacePath = ""
    @Published var showsTaskFiles = false

    private let port: Int
    private var sessionID: String
    private var socket: URLSessionWebSocketTask?
    private var pendingMessages: [PendingNativeChatMessage] = []
    private var pendingMemorySummary: String?
    private var reconnectTask: Task<Void, Never>?
    private var responseWatchdogTask: Task<Void, Never>?
    private var lastSentTextRequest: (content: String, modelOverride: String)?
    private var reconnectAttempt = 0
    private var connectionEpoch = 0

    var isResponseInProgress: Bool {
        isThinking || isExecutingPlan
    }

    init(port: Int) {
        self.port = port
        let workspaceURL = jameTaskFolderURL().standardizedFileURL
        workspaceName = workspaceURL.lastPathComponent
        workspacePath = workspaceURL.path
        let key = "jameclaw.native-chat.session-id"
        // The native window does not restore the old message transcript on
        // launch, so reusing its old gateway session would silently inject
        // invisible history into the first new message.
        let newID = UUID().uuidString
        UserDefaults.standard.set(newID, forKey: key)
        sessionID = newID
    }

    func startGatewayAndConnect() {
        let epoch = connectionEpoch
        Task {
            do {
                try await restoreInternalWorkspaceIfNeeded()
                let request = authenticatedConsoleRequest(port: port, path: "/api/gateway/start", method: "POST")
                let (_, response) = try await URLSession.shared.data(for: request)
                guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                    throw URLError(.badServerResponse)
                }
                try await waitForGatewayReadiness()
                guard epoch == connectionEpoch else { return }
                connect()
            } catch {
                guard epoch == connectionEpoch else { return }
                startBundledLauncherIfNeeded()
                reportError(title: "JameClaw is not ready", detail: connectionDetail(for: error))
                scheduleReconnect()
            }
        }
    }

    func connect() {
        guard socket == nil else { return }
        let epoch = connectionEpoch
        reconnectTask?.cancel()
        reconnectTask = nil
        status = "Connecting…"
        Task {
            do {
                let setup = authenticatedConsoleRequest(port: port, path: "/api/jame/setup", method: "POST")
                let (data, httpResponse) = try await URLSession.shared.data(for: setup)
                guard let http = httpResponse as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                    throw URLError(.badServerResponse)
                }
                let setupResponse = try JSONSerialization.jsonObject(with: data) as? [String: Any]
                guard let token = setupResponse?["token"] as? String,
                      let rawURL = setupResponse?["ws_url"] as? String,
                      var parts = URLComponents(string: rawURL) else { throw URLError(.badServerResponse) }
                var query = parts.queryItems ?? []
                query.append(URLQueryItem(name: "session_id", value: sessionID))
                parts.queryItems = query
                guard let wsURL = parts.url else { throw URLError(.badURL) }
                guard epoch == connectionEpoch else { return }
                let task = URLSession.shared.webSocketTask(with: wsURL, protocols: ["token.\(token)"])
                socket = task
                task.resume()
                // URLSession does not expose a WebSocket "open" callback.
                // A ping makes the upgrade observable before the UI accepts or
                // flushes queued messages, matching the Web Console's onopen
                // behavior rather than showing a false-ready state.
                let pingID = "native-ping-\(UUID().uuidString)"
                let ping = try JSONSerialization.data(withJSONObject: ["type": "ping", "id": pingID])
                guard let pingText = String(data: ping, encoding: .utf8) else { throw URLError(.badServerResponse) }
                try await task.send(.string(pingText))
                guard epoch == connectionEpoch else {
                    task.cancel(with: .goingAway, reason: nil)
                    return
                }
                status = "Ready"
                lastError = nil
                loadDisplayName()
                reconnectAttempt = 0
                flushPendingMessages()
                receive(from: task, epoch: epoch)
            } catch {
                guard epoch == connectionEpoch else { return }
                socket = nil
                startBundledLauncherIfNeeded()
                reportError(title: "Could not connect to JameClaw", detail: connectionDetail(for: error))
                scheduleReconnect()
            }
        }
    }

    private func scheduleReconnect() {
        guard reconnectTask == nil else { return }
        let delay = min(pow(2, Double(reconnectAttempt)), 15)
        reconnectAttempt = min(reconnectAttempt + 1, 4)
        status = "Reconnecting…"
        reconnectTask = Task { [weak self] in
            try? await Task.sleep(for: .seconds(delay))
            guard !Task.isCancelled else { return }
            self?.reconnectTask = nil
            self?.startGatewayAndConnect()
        }
    }

    func retryConnection() {
        connectionEpoch += 1
        reconnectTask?.cancel()
        reconnectTask = nil
        socket?.cancel(with: .goingAway, reason: nil)
        socket = nil
        lastError = nil
        reconnectAttempt = 0
        startGatewayAndConnect()
    }

    func dismissError() { lastError = nil }

    var canRetryLastMessage: Bool { lastSentTextRequest != nil }
    var canContinueConversation: Bool { messages.contains { $0.role == "user" || $0.role == "assistant" } }

    func retryLastMessage() {
        guard let request = lastSentTextRequest else {
            retryConnection()
            return
        }
        lastError = nil
        isThinking = true
        status = socket == nil ? "Reconnecting to retry…" : "Retrying message…"
        let id = "native-retry-\(UUID().uuidString)"
        armResponseWatchdog(for: id)
        if socket == nil {
            pendingMessages.append(PendingNativeChatMessage(
                id: id,
                content: request.content,
                modelOverride: request.modelOverride
            ))
            connect()
            return
        }
        send(id: id, content: request.content, modelOverride: request.modelOverride)
    }

	func continueConversation(modelOverride: String = "") {
        lastError = nil
        draft = "Continue from where you stopped. Keep the work already completed and finish the remaining task."
		send(modelOverride: modelOverride)
	}

	/// Sends a control command from the native status bar through the same
	/// authenticated session as normal chat. Keeping these actions on the
	/// shared command path also makes the native UI and Telegram consistent.
	func sendControlCommand(_ command: String) {
		let value = command.trimmingCharacters(in: .whitespacesAndNewlines)
		guard !value.isEmpty else { return }
		let commandName = value.split(separator: " ").first.map(String.init)?.lowercased() ?? ""
		guard !isResponseInProgress || ["/stop", "/abort", "/cancel"].contains(commandName) else { return }
		draft = value
		send()
	}

    private func loadDisplayName() {
        Task {
            guard let data = try? await URLSession.shared.data(
                from: authenticatedConsoleURL(port: port, path: "/api/agents")
            ).0,
            let agents = try? JSONDecoder().decode(NativeAgentsResponse.self, from: data).agents,
            let main = agents.first(where: { $0.id == "main" }),
            !main.name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
            agentName = main.name
        }
    }

    func startNewChat() {
        let newID = UUID().uuidString
        switchConversation(to: newID, messages: [], statusText: "Starting a new conversation…")
    }

    fileprivate func resumeSession(_ request: NativeSessionResumeRequest) {
        let restored = request.messages.enumerated().map { index, message in
            NativeChatMessage(
                id: "restored-\(request.sessionID)-\(index)",
                role: message.role,
                content: message.content
            )
        }
        let suffix = request.cloned ? " from another channel" : ""
        switchConversation(
            to: request.sessionID,
            messages: restored,
            statusText: "Continuing \(request.title)\(suffix)…"
        )
    }

    private func switchConversation(
        to newSessionID: String,
        messages restoredMessages: [NativeChatMessage],
        statusText: String
    ) {
        connectionEpoch += 1
        reconnectTask?.cancel()
        reconnectTask = nil
        cancelResponseWatchdog()
        let previousSocket = socket
        socket = nil
        previousSocket?.cancel(with: .goingAway, reason: nil)
        sessionID = newSessionID
        UserDefaults.standard.set(newSessionID, forKey: "jameclaw.native-chat.session-id")
		messages = restoredMessages
		activity.removeAll()
        pendingMessages.removeAll()
        pendingMemorySummary = nil
        lastSentTextRequest = nil
        draft = ""
        isThinking = false
        isExecutingPlan = false
        lastError = nil
        reconnectAttempt = 0
        status = statusText
        startGatewayAndConnect()
    }

    func addFastPathExchange(user: String, response: String) {
        messages.append(NativeChatMessage(id: "local-user-\(UUID().uuidString)", role: "user", content: user))
        messages.append(NativeChatMessage(id: "local-response-\(UUID().uuidString)", role: "assistant", content: response))
        isThinking = false
        status = socket == nil ? "Connecting…" : "Ready"
    }

    func setWorkspace(_ workspaceURL: URL) {
        let selectedWorkspaceURL = workspaceURL.standardizedFileURL.resolvingSymlinksInPath()
        let resolvedWorkspaceURL = organizedJameTaskFolder(for: selectedWorkspaceURL)
        let resolvedWorkspacePath = resolvedWorkspaceURL.path
        Task {
            do {
                try FileManager.default.createDirectory(at: resolvedWorkspaceURL, withIntermediateDirectories: true)
                status = "Using \(resolvedWorkspaceURL.lastPathComponent) as the task folder…"
                try await saveTaskFolderAccess(
                    resolvedWorkspacePath,
                    legacyFolderPath: selectedWorkspaceURL.path == resolvedWorkspacePath ? nil : selectedWorkspaceURL.path,
                    restoreInternalWorkspace: true
                )
                UserDefaults.standard.set(resolvedWorkspacePath, forKey: jameTaskFolderDefaultsKey)

                let restart = authenticatedConsoleRequest(port: port, path: "/api/gateway/restart", method: "POST")
                let (_, restartResponse) = try await URLSession.shared.data(for: restart)
                guard let http = restartResponse as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                    throw URLError(.badServerResponse)
                }

                workspaceName = resolvedWorkspaceURL.lastPathComponent
                workspacePath = resolvedWorkspacePath
                status = "Task folder set. Reconnecting…"
                connectionEpoch += 1
                socket?.cancel(with: .goingAway, reason: nil)
                socket = nil
                lastError = nil
                scheduleReconnect()
            } catch {
                reportError(title: "Could not change workspace", detail: connectionDetail(for: error))
            }
        }
    }

    private func restoreInternalWorkspaceIfNeeded() async throws {
        let configuredWorkspace = jameWorkspaceURL().standardizedFileURL.resolvingSymlinksInPath()
        let internalWorkspace = jameInternalWorkspaceURL().resolvingSymlinksInPath()
        let legacyTaskFolder = configuredWorkspace.path == internalWorkspace.path
            ? jameTaskFolderURL().standardizedFileURL.resolvingSymlinksInPath()
            : configuredWorkspace
        let organizedTaskFolder = organizedJameTaskFolder(for: legacyTaskFolder)
        try FileManager.default.createDirectory(at: organizedTaskFolder, withIntermediateDirectories: true)

        UserDefaults.standard.set(organizedTaskFolder.path, forKey: jameTaskFolderDefaultsKey)
        workspaceName = organizedTaskFolder.lastPathComponent
        workspacePath = organizedTaskFolder.path
        try await saveTaskFolderAccess(
            organizedTaskFolder.path,
            legacyFolderPath: legacyTaskFolder.path == organizedTaskFolder.path ? nil : legacyTaskFolder.path,
            restoreInternalWorkspace: true
        )

        if configuredWorkspace.path != internalWorkspace.path || legacyTaskFolder.path != organizedTaskFolder.path {
            let restart = authenticatedConsoleRequest(port: port, path: "/api/gateway/restart", method: "POST")
            let (_, response) = try await URLSession.shared.data(for: restart)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
        }
    }

    private func saveTaskFolderAccess(
        _ folderPath: String,
        legacyFolderPath: String? = nil,
        restoreInternalWorkspace: Bool
    ) async throws {
        let configURL = FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent(".jameclaw/config.json")
        let localConfig = (try? Data(contentsOf: configURL))
            .flatMap { try? JSONSerialization.jsonObject(with: $0) as? [String: Any] }
        let tools = localConfig?["tools"] as? [String: Any]
        var readPaths = tools?["allow_read_paths"] as? [String] ?? []
        var writePaths = tools?["allow_write_paths"] as? [String] ?? []
        if let legacyFolderPath {
            readPaths.removeAll { $0 == legacyFolderPath }
            writePaths.removeAll { $0 == legacyFolderPath }
        }
        if !readPaths.contains(folderPath) { readPaths.append(folderPath) }
        if !writePaths.contains(folderPath) { writePaths.append(folderPath) }

        var patch: [String: Any] = [
            "tools": [
                "allow_read_paths": readPaths,
                "allow_write_paths": writePaths,
            ],
        ]
        if restoreInternalWorkspace {
            patch["agents"] = ["defaults": ["workspace": jameInternalWorkspaceURL().path]]
        }

        var request = authenticatedConsoleRequest(port: port, path: "/api/config", method: "PATCH")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONSerialization.data(withJSONObject: patch)
        let (_, response) = try await URLSession.shared.data(for: request)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw URLError(.badServerResponse)
        }
    }

    func reflectWorkspace(_ workspaceURL: URL) {
        let resolved = workspaceURL.standardizedFileURL.resolvingSymlinksInPath()
        workspaceName = resolved.lastPathComponent
        workspacePath = resolved.path
    }

    // The top-level desktop app owns the visible window. Start its bundled
    // backend helper when needed so Finder and Dock launches are self-contained.
    private func startBundledLauncherIfNeeded() {
        NativeLaunchCoordinator.shared.startIfNeeded()
    }

    func send(modelOverride: String = "") {
        let content = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !content.isEmpty else { return }
        if mentionsTaskFiles(in: content) {
            showsTaskFiles = true
        }
        let wasBusy = isResponseInProgress
        draft = ""
        let id = "native-\(UUID().uuidString)"
        messages.append(NativeChatMessage(id: id, role: "user", content: content))
        if wasBusy { queuedMessageCount += 1 }
        isThinking = true
        armResponseWatchdog(for: id)
        let outboundContent = taskFolderInstruction(for: nativeAppCommandInstruction(for: content))
        lastSentTextRequest = (outboundContent, modelOverride)
        guard socket != nil else {
            pendingMessages.append(PendingNativeChatMessage(id: id, content: outboundContent, modelOverride: modelOverride))
            status = "Connecting…"
            connect()
            return
        }
        send(id: id, content: outboundContent, modelOverride: modelOverride)
    }

    func sendSkillImported(_ skillName: String) {
        draft = "I uploaded the \(skillName) skill to this workspace. Please read and use it for this task."
        send()
    }

    private func flushPendingMessages() {
        let queued = pendingMessages
        pendingMessages.removeAll()
        for message in queued {
            send(id: message.id, content: message.content, modelOverride: message.modelOverride)
        }
    }

    private func send(id: String, content: String, modelOverride: String = "") {
        var payload: [String: Any] = ["content": content]
        if !modelOverride.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            payload["model"] = modelOverride.trimmingCharacters(in: .whitespacesAndNewlines)
        }
        let envelope: [String: Any] = [
            "type": "message.send",
            "id": id,
            "session_id": sessionID,
            "payload": payload,
        ]
        guard let data = try? JSONSerialization.data(withJSONObject: envelope),
              let text = String(data: data, encoding: .utf8) else {
            fail(messageID: id, message: "Could not prepare that message.")
            return
        }
        Task {
            do {
                guard let socket else { throw URLError(.notConnectedToInternet) }
                try await socket.send(.string(text))
            } catch {
                fail(messageID: id, message: "Message failed to send. Reconnecting…")
                socket = nil
                connect()
            }
        }
    }

    private func taskFolderInstruction(for content: String) -> String {
        let folderPath = workspacePath.trimmingCharacters(in: .whitespacesAndNewlines)
        guard showsTaskFiles, !folderPath.isEmpty, folderPath != jameInternalWorkspaceURL().path else { return content }
        return """
        [JameClaw Desktop task folder: \(folderPath)]
        Treat this folder as the working directory for this request. Use absolute paths inside it when reading, creating, editing, or running project files. For multi-file work that does not already belong to a named project, create one clearly named project subfolder here. Never create JameClaw runtime folders such as memory, sessions, cron, skills, or artifacts here; those belong only in JameClaw's private internal workspace.

        \(content)
        """
    }

    private func mentionsTaskFiles(in content: String) -> Bool {
        let normalized = content.lowercased()
        let terms = [
            "task file", "task files", "task folder", "working folder",
            "project folder", "workspace folder", "workspace", "file",
            "folder", ".txt", ".md", ".csv", ".json", ".swift", ".go",
            ".tsx", ".ts", ".js", ".py", ".html", ".css"
        ]
        return terms.contains { normalized.contains($0) }
    }

    func sendMedia(data: Data, filename: String, contentType: String, kind: String, content: String = "", modelOverride: String = "") {
        lastSentTextRequest = nil
        if mentionsTaskFiles(in: content) || mentionsTaskFiles(in: filename) {
            showsTaskFiles = true
        }
        guard socket != nil else {
            status = "Connecting to Jame. Try the upload again in a moment."
            connect()
            return
        }
        let id = "\(kind)-\(UUID().uuidString)"
        let displayContent = content.isEmpty ? "📎 \(filename)" : "\(content)\n📎 \(filename)"
        messages.append(NativeChatMessage(id: id, role: "user", content: displayContent))
        isThinking = true
        armResponseWatchdog(for: id)
        var payload: [String: Any] = [
            "content": taskFolderInstruction(for: content),
            "data": data.base64EncodedString(),
            "filename": filename,
            "content_type": contentType,
            "kind": kind,
        ]
        if !modelOverride.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            payload["model"] = modelOverride.trimmingCharacters(in: .whitespacesAndNewlines)
        }
        let envelope: [String: Any] = [
            "type": "media.send",
            "id": id,
            "session_id": sessionID,
            "payload": payload,
        ]
        guard let encoded = try? JSONSerialization.data(withJSONObject: envelope),
              let text = String(data: encoded, encoding: .utf8) else {
            fail(messageID: id, message: "Could not prepare that upload.")
            return
        }
        Task {
            do {
                guard let socket else { throw URLError(.notConnectedToInternet) }
                try await socket.send(.string(text))
            } catch {
                fail(messageID: id, message: "Upload failed. Reconnecting…")
                socket = nil
                connect()
            }
        }
    }

    private func fail(messageID: String, message: String) {
        messages.append(NativeChatMessage(id: "error-\(UUID().uuidString)", role: "error", content: message))
        reportError(title: "Message error", detail: message)
    }

    private func reportError(title: String, detail: String) {
        cancelResponseWatchdog()
        status = title
        isThinking = false
        isExecutingPlan = false
        removeExecutionPlanMessages()
        lastError = NativeAppError(title: title, detail: detail)
    }

    private func connectionDetail(for error: Error) -> String {
        if let urlError = error as? URLError {
            switch urlError.code {
            case .cannotConnectToHost, .networkConnectionLost, .notConnectedToInternet:
                return "Check that the JameClaw launcher is running, then try again."
            case .timedOut:
                return "The launcher took too long to respond. It may still be starting."
            default:
                break
            }
        }
        return error.localizedDescription.isEmpty ? "The launcher returned an unexpected response." : error.localizedDescription
    }

    private func waitForGatewayReadiness() async throws {
        for _ in 0..<20 {
            let (data, response) = try await URLSession.shared.data(
                for: authenticatedConsoleRequest(port: port, path: "/api/gateway/status")
            )
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            let state = try JSONSerialization.jsonObject(with: data) as? [String: Any]
            if state?["gateway_status"] as? String == "running" {
                return
            }
            if state?["gateway_status"] as? String == "error" {
                let reason = state?["gateway_start_reason"] as? String ?? "The gateway could not start. Check the configured chat model."
                throw NativeGatewayError.notReady(reason)
            }
            try await Task.sleep(for: .milliseconds(500))
        }
        throw NativeGatewayError.notReady("The gateway is still starting. It will retry automatically.")
    }

    private func receive(from task: URLSessionWebSocketTask, epoch: Int) {
        Task {
            do {
                let message = try await task.receive()
                guard epoch == connectionEpoch, socket === task else { return }
                if case let .string(text) = message { handle(text) }
                receive(from: task, epoch: epoch)
            } catch {
                guard epoch == connectionEpoch, socket === task else { return }
                socket = nil
                reportError(title: "Connection lost", detail: "Jame will retry automatically. You can also retry now.")
                scheduleReconnect()
            }
        }
    }

    private func handle(_ text: String) {
        guard let data = text.data(using: .utf8),
              let event = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let type = event["type"] as? String else { return }
        let payload = event["payload"] as? [String: Any] ?? [:]
        switch type {
		case "typing.start": isThinking = true
        case "typing.stop":
            isThinking = false
        case "message.create":
            let content = responseContent(from: payload)
            let id = (payload["message_id"] as? String) ?? (event["id"] as? String) ?? UUID().uuidString
            if nativePlanSteps(from: content) != nil {
                isExecutingPlan = true
                isThinking = true
            } else if isThinkingPlaceholder(content) {
                isThinking = true
            } else {
                removeExecutionPlanMessages()
                isExecutingPlan = false
                isThinking = false
                cancelResponseWatchdog()
            }
            upsertAssistantMessage(id: id, content: content)
            if !isThinkingPlaceholder(content), nativePlanSteps(from: content) == nil {
                lastSentTextRequest = nil
            }
        case "message.update":
            let id = (payload["message_id"] as? String) ?? (event["id"] as? String) ?? UUID().uuidString
            let content = responseContent(from: payload)
            if nativePlanSteps(from: content) != nil {
                isExecutingPlan = true
                isThinking = true
            } else if isThinkingPlaceholder(content) {
                isThinking = true
            } else if !content.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                removeExecutionPlanMessages()
                isExecutingPlan = false
                // Streaming final text is already a response, but keep the
                // send control busy until typing.stop or message.create.
                isThinking = true
            }
            // A gateway can begin streaming before its placeholder reaches the
            // desktop. Treat that update as the first visible assistant reply
            // instead of dropping it.
            upsertAssistantMessage(id: id, content: content)
        case "plan.update":
            applyPlanUpdate(payload)
		case "memory.changed":
            let summary = (payload["summary"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines)
            pendingMemorySummary = summary?.isEmpty == false
                ? summary
                : "Jame updated memory for future conversations."
			appendPendingMemoryNoticeIfPossible()
		case "activity.update":
			let label = (payload["label"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
			guard !label.isEmpty else { return }
			activity.append(label)
			if activity.count > 8 { activity.removeFirst(activity.count - 8) }
		case "task.complete":
			queuedMessageCount = max(0, queuedMessageCount - 1)
            lastSentTextRequest = nil
            removeExecutionPlanMessages()
            isExecutingPlan = false
            isThinking = false
            cancelResponseWatchdog()
			appendPendingMemoryNoticeIfPossible()
			activity.removeAll()
            NotificationCenter.default.post(name: .jameclawAutomationChanged, object: nil)
            notifyTaskCompletion(responseContent(from: payload))
		case "error":
			queuedMessageCount = max(0, queuedMessageCount - 1)
            let message = responseContent(from: payload, fallback: "The provider could not complete this request.")
            messages.append(NativeChatMessage(id: UUID().uuidString, role: "error", content: message))
            reportError(title: "Provider error", detail: message)
        default: break
        }
    }

    /// A provider can leave a websocket open forever without ever returning a
    /// completion event. Give long-running agent work plenty of room, then
    /// return control to the user with explicit Retry and Continue actions.
    private func armResponseWatchdog(for messageID: String) {
        responseWatchdogTask?.cancel()
        responseWatchdogTask = Task { [weak self] in
            try? await Task.sleep(for: .seconds(8 * 60))
            guard !Task.isCancelled,
                  let self,
                  self.isResponseInProgress else { return }
            self.responseWatchdogTask = nil
            self.reportError(
                title: "Response appears stuck",
                detail: "Jame has not completed this response after 8 minutes. Retry the message or continue the conversation."
            )
            NSLog("[JameChat] response watchdog fired for %@", messageID)
        }
    }

    private func cancelResponseWatchdog() {
        responseWatchdogTask?.cancel()
        responseWatchdogTask = nil
    }

    private func responseContent(from payload: [String: Any], fallback: String = "") -> String {
        for key in ["content", "text", "message", "error"] {
            if let value = payload[key] as? String, !value.isEmpty { return value }
        }
        return fallback
    }

    private func isThinkingPlaceholder(_ content: String) -> Bool {
        let normalized = content
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        return normalized == "thinking"
            || normalized == "thinking..."
            || normalized == "thinking…"
            || (normalized.contains("thinking") && normalized.contains("💭"))
    }

    private func removeExecutionPlanMessages() {
        messages.removeAll { message in
            message.role == "assistant" && nativePlanSteps(from: message.content) != nil
        }
    }

    private func upsertAssistantMessage(id: String, content: String) {
        if let index = messages.firstIndex(where: { $0.id == id }) {
            messages[index].content = content
        } else {
            messages.append(NativeChatMessage(id: id, role: "assistant", content: content))
        }
        appendPendingMemoryNoticeIfPossible()
    }

    private func appendPendingMemoryNoticeIfPossible() {
        guard let summary = pendingMemorySummary else { return }
        let lastUserIndex = messages.lastIndex(where: { $0.role == "user" }) ?? -1
        guard messages.indices.contains(where: { index in
            index > lastUserIndex && messages[index].role == "assistant"
        }) else { return }
        messages.append(NativeChatMessage(
            id: "memory-change-\(UUID().uuidString)",
            role: "memory",
            content: summary
        ))
        pendingMemorySummary = nil
    }

    private func applyPlanUpdate(_ payload: [String: Any]) {
        let planIndex = messages.lastIndex(where: {
            $0.role == "assistant" && nativePlanSteps(from: $0.content) != nil
        })
        var steps = planIndex.flatMap { nativePlanSteps(from: messages[$0].content) } ?? []

        if payload["complete"] as? Bool == true {
            guard !steps.isEmpty else { return }
            for index in steps.indices where steps[index].status != .cancelled {
                steps[index].status = .completed
            }
        } else if let todos = payload["todos"] as? [[String: Any]], !todos.isEmpty {
            var updated: [NativePlanStep] = []
            for (index, todo) in todos.enumerated() {
                let current = steps.indices.contains(index) ? steps[index] : nil
                let status = NativePlanStepStatus(rawValue: todo["status"] as? String ?? "pending") ?? .pending
                let title = current?.title ?? (todo["content"] as? String ?? "Step \(index + 1)")
                updated.append(NativePlanStep(
                    id: current?.id ?? "plan-step-\(index)",
                    title: title,
                    tools: current?.tools ?? [],
                    status: status
                ))
            }
            steps = updated
        } else {
            return
        }

        isExecutingPlan = true
        isThinking = true

        let content = nativePlanContent(from: steps)
        if let planIndex {
            messages[planIndex].content = content
        } else {
            messages.append(NativeChatMessage(
                id: "plan-\(UUID().uuidString)",
                role: "assistant",
                content: content
            ))
        }
    }

    private func notifyTaskCompletion(_ result: String) {
        let key = "jame.notifications.taskCompletion"
        guard !NSApp.isActive,
              UserDefaults.standard.object(forKey: key) == nil || UserDefaults.standard.bool(forKey: key) else {
            return
        }
        let content = UNMutableNotificationContent()
        content.title = "\(agentName) finished the task"
        let cleanResult = result
            .replacingOccurrences(of: "**", with: "")
            .replacingOccurrences(of: "\n", with: " ")
            .trimmingCharacters(in: .whitespacesAndNewlines)
        content.body = cleanResult.isEmpty ? "Open JameClaw Desktop to see the result." : cleanResult
        content.sound = .default
        UNUserNotificationCenter.current().add(UNNotificationRequest(
            identifier: "jame-task-\(UUID().uuidString)",
            content: content,
            trigger: nil
        ))
    }
}

private struct NativeActivityRail: View {
	let activity: [String]
	let accent: Color
	let fontScale: CGFloat
	@State private var isExpanded = true

	var body: some View {
		if !activity.isEmpty {
			VStack(alignment: .leading, spacing: 6) {
				HStack(spacing: 7) {
					Button { isExpanded.toggle() } label: {
						Image(systemName: isExpanded ? "chevron.down" : "chevron.right")
							.font(.system(size: 9 * fontScale, weight: .bold))
						Circle().fill(accent).frame(width: 6, height: 6)
						Text("LIVE ACTIVITY")
					}
					.buttonStyle(.plain)
						.font(.system(size: 10 * fontScale, weight: .semibold, design: .monospaced))
						.foregroundStyle(accent)
					Spacer()
					Text("latest")
						.font(.system(size: 10 * fontScale, design: .monospaced))
						.foregroundStyle(.secondary)
				}
				if isExpanded {
					ForEach(Array(activity.suffix(3).enumerated()), id: \.offset) { _, item in
						Text(item)
							.font(.system(size: 11 * fontScale, design: .monospaced))
							.foregroundStyle(.secondary)
							.lineLimit(1)
					}
				}
			}
			.padding(10)
			.background(accent.opacity(0.07), in: Rectangle())
			.overlay(alignment: .leading) { Rectangle().fill(accent).frame(width: 2) }
		}
	}
}

private struct NativeAgentControlBar: View {
	@ObservedObject var chat: NativeChatStore
	let accent: Color
	let fontScale: CGFloat
	let modelName: String
	let providerName: String

	var body: some View {
		HStack(spacing: 8) {
			Image(systemName: chat.isResponseInProgress ? "bolt.fill" : "checkmark.circle.fill")
				.foregroundStyle(chat.isResponseInProgress ? accent : .green)
			Text(chat.isResponseInProgress ? "WORKING" : "READY")
				.font(.system(size: 10 * fontScale, weight: .semibold, design: .monospaced))
				.foregroundStyle(.secondary)
			Text("· \(chat.messages.count) messages")
				.font(.system(size: 10 * fontScale, design: .monospaced))
				.foregroundStyle(.secondary)
			if !modelName.isEmpty {
				Text("· \(providerName.isEmpty ? "Auto" : providerName) / \(modelName)")
					.font(.system(size: 10 * fontScale, design: .monospaced))
					.foregroundStyle(.secondary)
					.lineLimit(1)
			}
			Spacer()
			if chat.isResponseInProgress {
				Button("Stop", systemImage: "stop.fill") { chat.sendControlCommand("/stop") }
					.tint(.red)
			}
			if chat.queuedMessageCount > 0 {
				Text("· \(chat.queuedMessageCount) queued")
					.font(.system(size: 10 * fontScale, design: .monospaced))
					.foregroundStyle(accent)
			}
			Button("Status") { chat.sendControlCommand("/status") }
			Button("Undo") { chat.sendControlCommand("/undo") }
			Button("Compact") { chat.sendControlCommand("/compact") }
			Button("New chat") { chat.startNewChat() }
		}
		.font(.caption)
		.buttonStyle(.bordered)
		.controlSize(.small)
		.disabled(false)
		.padding(.horizontal, 14)
		.padding(.vertical, 8)
		.background(accent.opacity(0.045))
	}
}

private enum NativeGatewayError: LocalizedError {
    case notReady(String)

    var errorDescription: String? {
        switch self {
        case let .notReady(message): return message
        }
    }
}

@MainActor
private final class NativeTerminalStore: ObservableObject {
    @Published var output = ""
    @Published var command = ""
    @Published var isRunning = false
    @Published var workingDirectoryURL = jameWorkspaceURL()

    private var process: Process?
    private var inputPipe: Pipe?
    private var outputPipe: Pipe?

    func start() {
        guard process == nil else { return }

        let shell = ProcessInfo.processInfo.environment["SHELL"] ?? "/bin/zsh"
        let process = Process()
        let input = Pipe()
        let output = Pipe()
        process.executableURL = URL(fileURLWithPath: shell)
        process.arguments = ["-l"]
        process.currentDirectoryURL = workingDirectoryURL
        process.standardInput = input
        process.standardOutput = output
        process.standardError = output
        process.terminationHandler = { [weak self] _ in
            DispatchQueue.main.async {
                self?.process = nil
                self?.inputPipe = nil
                self?.outputPipe?.fileHandleForReading.readabilityHandler = nil
                self?.outputPipe = nil
                self?.isRunning = false
            }
        }
        output.fileHandleForReading.readabilityHandler = { [weak self] handle in
            let data = handle.availableData
            guard !data.isEmpty, let text = String(data: data, encoding: .utf8) else { return }
            DispatchQueue.main.async {
                self?.append(text)
            }
        }

        do {
            try process.run()
            self.process = process
            inputPipe = input
            outputPipe = output
            isRunning = true
            append("JameClaw terminal — \(workingDirectoryURL.path)\n")
        } catch {
            append("Could not start terminal: \(error.localizedDescription)\n")
        }
    }

    func send() {
        let value = command.trimmingCharacters(in: .newlines)
        guard !value.isEmpty else { return }
        start()
        guard let data = (value + "\n").data(using: .utf8) else { return }
        append("$ \(value)\n")
        inputPipe?.fileHandleForWriting.write(data)
        command = ""
    }

    func clear() { output = "" }

    func useWorkingDirectory(_ url: URL) {
        let directory = url.standardizedFileURL.resolvingSymlinksInPath()
        workingDirectoryURL = directory
        guard isRunning else {
            start()
            return
        }
        let escapedPath = directory.path.replacingOccurrences(of: "'", with: "'\\''")
        guard let data = "cd '\(escapedPath)'\npwd\n".data(using: .utf8) else { return }
        inputPipe?.fileHandleForWriting.write(data)
        append("\nWorkspace changed — \(directory.path)\n")
    }

    func stop() {
        process?.terminate()
    }

    private func append(_ text: String) {
        output += text
        // Keep the drawer responsive when a command emits a lot of output.
        if output.count > 160_000 {
            output = String(output.suffix(120_000))
        }
    }

    deinit {
        outputPipe?.fileHandleForReading.readabilityHandler = nil
        process?.terminate()
    }
}

private struct NativeTerminalPanel: View {
    @ObservedObject var terminal: NativeTerminalStore
    let theme: LauncherTheme
    let accent: Color
    let fontScale: Double

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 8) {
                Image(systemName: "terminal.fill")
                    .foregroundStyle(accent)
                Text("Terminal")
                    .font(.headline)
                Spacer()
                Button { terminal.clear() } label: {
                    Image(systemName: "trash")
                }
                .buttonStyle(.borderless)
                .help("Clear terminal output")
                Button { terminal.stop() } label: {
                    Image(systemName: "stop.fill")
                        .foregroundStyle(.red)
                }
                .buttonStyle(.borderless)
                .disabled(!terminal.isRunning)
                .help("Stop terminal")
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 12)
            .background(theme.panel)

            ScrollViewReader { proxy in
                ScrollView {
                    Text(terminal.output.isEmpty ? "Terminal ready. Enter a command below." : terminal.output)
                        .font(.system(size: 12 * fontScale, design: .monospaced))
                        .foregroundStyle(theme.text)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .textSelection(.enabled)
                        .id("terminal-output")
                        .padding(14)
                }
                .onChange(of: terminal.output) { _, _ in
                    proxy.scrollTo("terminal-output", anchor: .bottom)
                }
            }
            .background(theme.background)

            HStack(spacing: 8) {
                Text("$")
                    .font(.system(size: 14 * fontScale, weight: .semibold, design: .monospaced))
                    .foregroundStyle(accent)
                TextField("Run a command", text: $terminal.command)
                    .font(.system(size: 13 * fontScale, design: .monospaced))
                    .textFieldStyle(.plain)
                    .onSubmit { terminal.send() }
                Button("Run") { terminal.send() }
                    .buttonStyle(.borderedProminent)
                    .tint(accent)
            }
            .padding(12)
            .background(theme.panel)
        }
        .onAppear { terminal.start() }
    }
}

@MainActor
private final class NativeDocumentIndexStore: ObservableObject {
    @Published var entries: [WorkspaceEntry] = []
    @Published var isLoading = false
    @Published var status = ""

    let roots: [URL]

    init() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let candidates = [
            jameWorkspaceURL(),
            home.appendingPathComponent("Desktop", isDirectory: true),
            home.appendingPathComponent("Documents", isDirectory: true),
            home.appendingPathComponent("Downloads", isDirectory: true),
        ]
        var seen = Set<String>()
        roots = candidates.filter {
            FileManager.default.fileExists(atPath: $0.path) && seen.insert($0.standardizedFileURL.path).inserted
        }
        status = "Document list waits for your safety approval."
    }

    func refresh(includeAllLocations: Bool = true) {
        guard !isLoading else { return }
        isLoading = true
        status = "Scanning documents…"
        let roots = includeAllLocations ? roots : [jameWorkspaceURL()]
        Task {
            let loaded = await Task.detached(priority: .userInitiated) {
                Self.scan(roots: roots)
            }.value
            entries = loaded
            status = loaded.count >= 1_500
                ? "Showing the first 1,500 documents and folders. Search or choose another item."
                : "\(loaded.count) documents and folders"
            isLoading = false
        }
    }

    nonisolated private static func scan(roots: [URL]) -> [WorkspaceEntry] {
        var found: [WorkspaceEntry] = []
        var seen = Set<String>()
        for root in roots {
            guard found.count < 1_500,
                  let enumerator = FileManager.default.enumerator(
                    at: root,
                    includingPropertiesForKeys: [.isDirectoryKey],
                    options: [.skipsHiddenFiles, .skipsPackageDescendants],
                    errorHandler: { _, _ in true }
                  ) else { continue }
            for case let url as URL in enumerator {
                guard found.count < 1_500 else { break }
                let path = url.standardizedFileURL.path
                guard seen.insert(path).inserted else { continue }
                let values = try? url.resourceValues(forKeys: [.isDirectoryKey])
                found.append(WorkspaceEntry(url: url, isDirectory: values?.isDirectory ?? false))
            }
        }
        return found.sorted {
            if $0.isDirectory != $1.isDirectory { return $0.isDirectory }
            return $0.url.lastPathComponent.localizedStandardCompare($1.url.lastPathComponent) == .orderedAscending
        }
    }
}

private struct TerminalWorkspaceView: View {
    let port: Int
    @Binding var isPresented: Bool
    @StateObject private var terminal = NativeTerminalStore()
    @StateObject private var documents = NativeDocumentIndexStore()
    @AppStorage("launcher.design.theme") private var savedTheme = LauncherTheme.light.rawValue
    @Environment(\.colorScheme) private var systemColorScheme
    @State private var search = ""
    @State private var selectedPath = ""
    @State private var workspaceStatus = "Choose a folder or document to tell Jame where to work."

    private var filteredEntries: [WorkspaceEntry] {
        let query = search.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !query.isEmpty else { return documents.entries }
        return documents.entries.filter {
            $0.url.lastPathComponent.localizedCaseInsensitiveContains(query)
                || $0.url.path.localizedCaseInsensitiveContains(query)
        }
    }
    private var theme: LauncherTheme {
        launcherThemePreference(from: savedTheme).resolved(for: systemColorScheme)
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 10) {
                Image(systemName: "terminal.fill").foregroundStyle(JameBrand.orange)
                VStack(alignment: .leading, spacing: 2) {
                    Text("Terminal + Documents").font(.headline)
                    Text("Select an item to set the agent workspace and terminal location.")
                        .font(.caption).foregroundStyle(JameBrand.muted)
                }
                Spacer()
                Button("Done") { isPresented = false }
                    .buttonStyle(.bordered)
            }
            .padding(16)
            .background(JameBrand.panel)

            HSplitView {
                VStack(spacing: 0) {
                    HStack(spacing: 8) {
                        Image(systemName: "magnifyingglass").foregroundStyle(JameBrand.orange)
                        TextField("Search documents and folders", text: $search)
                            .textFieldStyle(.plain)
                        Button { loadDocumentIndex() } label: { Image(systemName: "arrow.clockwise") }
                            .buttonStyle(.plain)
                            .help("Refresh documents")
                    }
                    .padding(10)
                    .background(JameBrand.elevated)
                    .overlay(Rectangle().stroke(JameBrand.rule, lineWidth: 1))

                    ScrollView {
                        LazyVStack(alignment: .leading, spacing: 0) {
                            Text("WORKSPACE ROOTS")
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(JameBrand.muted)
                                .padding(.horizontal, 12).padding(.top, 12).padding(.bottom, 6)
                            ForEach(documents.roots, id: \.path) { root in
                                documentButton(WorkspaceEntry(url: root, isDirectory: true), isRoot: true)
                            }

                            Text("DOCUMENTS AND FOLDERS")
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(JameBrand.muted)
                                .padding(.horizontal, 12).padding(.top, 16).padding(.bottom, 6)
                            ForEach(filteredEntries) { entry in
                                documentButton(entry, isRoot: false)
                            }
                        }
                    }

                    VStack(alignment: .leading, spacing: 8) {
                        Text(workspaceStatus)
                            .font(.caption)
                            .foregroundStyle(JameBrand.muted)
                            .lineLimit(2)
                        HStack {
                            Text(documents.status).font(.caption2).foregroundStyle(.secondary)
                            Spacer()
                            Button("Choose another…", action: chooseAnother)
                                .buttonStyle(.bordered)
                        }
                    }
                    .padding(12)
                    .background(JameBrand.panel)
                }
                .frame(minWidth: 230, idealWidth: 320, maxWidth: 430)

                NativeTerminalPanel(
                    terminal: terminal,
                    theme: theme,
                    accent: JameBrand.orange,
                    fontScale: 1
                )
                .frame(minWidth: 360)
            }
        }
        .frame(minWidth: 680, minHeight: 480)
        .background(JameBrand.ink)
        .tint(JameBrand.orange)
        .buttonBorderShape(.roundedRectangle(radius: 0))
        .onAppear { loadDocumentIndex() }
    }

    private func documentButton(_ entry: WorkspaceEntry, isRoot: Bool) -> some View {
        Button {
            use(entry)
        } label: {
            HStack(spacing: 9) {
                Image(systemName: entry.isDirectory ? "folder.fill" : "doc.text")
                    .foregroundStyle(entry.isDirectory ? JameBrand.orange : JameBrand.paper)
                    .frame(width: 18)
                VStack(alignment: .leading, spacing: 2) {
                    Text(entry.url.lastPathComponent)
                        .font(.subheadline.weight(isRoot ? .semibold : .regular))
                        .foregroundStyle(selectedPath == entry.url.path ? JameBrand.ink : JameBrand.paper)
                        .lineLimit(1)
                    Text(entry.url.deletingLastPathComponent().path)
                        .font(.caption2.monospaced())
                        .foregroundStyle(selectedPath == entry.url.path ? JameBrand.ink.opacity(0.7) : JameBrand.muted)
                        .lineLimit(1).truncationMode(.middle)
                }
                Spacer()
                Image(systemName: "arrow.right")
                    .font(.caption)
                    .foregroundStyle(selectedPath == entry.url.path ? JameBrand.ink : JameBrand.muted)
            }
            .padding(.horizontal, 12).padding(.vertical, 8)
            .frame(maxWidth: .infinity, alignment: .leading)
            .contentShape(Rectangle())
            .background(selectedPath == entry.url.path ? JameBrand.orange : Color.clear)
            .overlay(alignment: .bottom) { Rectangle().fill(JameBrand.rule).frame(height: 1) }
        }
        .buttonStyle(.plain)
    }

    private func chooseAnother() {
        let panel = NSOpenPanel()
        panel.title = "Choose where Jame should work"
        panel.message = "Choose a folder, or choose a document to use its containing folder as the agent workspace."
        panel.prompt = "Use with Jame"
        panel.canChooseFiles = true
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false
        guard panel.runModal() == .OK, let url = panel.url else { return }
        var isDirectory: ObjCBool = false
        FileManager.default.fileExists(atPath: url.path, isDirectory: &isDirectory)
        use(WorkspaceEntry(url: url, isDirectory: isDirectory.boolValue))
    }

    private func loadDocumentIndex() {
        let rawPolicy = UserDefaults.standard.string(forKey: "launcher.safety.documentApprovalPolicy")
            ?? DocumentApprovalPolicy.outsideWorkspace.rawValue
        let policy = DocumentApprovalPolicy(rawValue: rawPolicy) ?? .outsideWorkspace
        if policy == .workspaceOnly {
            documents.refresh(includeAllLocations: false)
            return
        }
        if policy == .explicitSelection {
            documents.refresh()
            return
        }
        let documentsURL = FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Documents", isDirectory: true)
        if approveDocumentAccess(to: documentsURL, action: "list document and folder names on this Mac") {
            documents.refresh()
        } else {
            documents.status = "Only the workspace is listed because broader document access was not approved."
            documents.refresh(includeAllLocations: false)
        }
    }

    private func use(_ entry: WorkspaceEntry) {
        let workspace = (entry.isDirectory ? entry.url : entry.url.deletingLastPathComponent())
            .standardizedFileURL.resolvingSymlinksInPath()
        guard approveDocumentAccess(
            to: entry.url,
            action: entry.isDirectory ? "use this folder as the agent workspace" : "work with this document"
        ) else { return }
        selectedPath = entry.url.path
        terminal.useWorkingDirectory(workspace)
        workspaceStatus = "Setting \(workspace.lastPathComponent) as Jame's workspace…"
        Task {
            do {
                try await persistWorkspace(workspace)
                workspaceStatus = entry.isDirectory
                    ? "Jame now works in \(workspace.path)"
                    : "Jame now works in \(workspace.path). The selected document is ready in Chat."
                NotificationCenter.default.post(
                    name: .jameclawWorkspaceChanged,
                    object: nil,
                    userInfo: [
                        "workspacePath": workspace.path,
                        "selectedPath": entry.url.path,
                        "isDirectory": entry.isDirectory,
                    ]
                )
            } catch {
                workspaceStatus = "Could not change workspace: \(error.localizedDescription)"
            }
        }
    }

    private func persistWorkspace(_ workspace: URL) async throws {
        var request = authenticatedConsoleRequest(port: port, path: "/api/config", method: "PATCH")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONSerialization.data(withJSONObject: [
            "agents": ["defaults": ["workspace": workspace.path]],
        ])
        let (_, response) = try await URLSession.shared.data(for: request)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw URLError(.badServerResponse)
        }

        let restart = authenticatedConsoleRequest(port: port, path: "/api/gateway/restart", method: "POST")
        let (_, restartResponse) = try await URLSession.shared.data(for: restart)
        guard let http = restartResponse as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw URLError(.badServerResponse)
        }
    }
}

private struct NativeSkillReference: Codable, Identifiable {
    let name: String
    let description: String
    let source: String
    let path: String
    var id: String { "skill-\(name)-\(source)" }
}

private struct NativeSkillsResponse: Codable {
    let skills: [NativeSkillReference]
}

private struct NativeSkillHealthWarning: Codable, Identifiable {
    let severity: String
    let code: String
    let message: String
    var id: String { "\(code)-\(message)" }
}

private struct NativeSkillHealth: Codable, Identifiable {
    let name: String
    let path: String
    let description: String
    let warnings: [NativeSkillHealthWarning]
    let usageCount: Int
    let successCount: Int
    let failureCount: Int
    let patchCount: Int
    var id: String { name }

    enum CodingKeys: String, CodingKey {
        case name, path, description, warnings
        case usageCount = "usage_count"
        case successCount = "success_count"
        case failureCount = "failure_count"
        case patchCount = "patch_count"
    }
}

private struct NativeFileReference: Codable, Identifiable {
    let name: String
    let path: String
    let directory: String
    var id: String { path }
}

private struct NativeFileSearchResponse: Codable {
    let items: [NativeFileReference]
}

private struct ChatComposerSuggestion: Identifiable {
	enum Kind { case app, skill, file, command }
    let id: String
    let kind: Kind
    let title: String
    let subtitle: String
    let insertion: String
    var icon: String {
        switch kind {
        case .app: return "app.badge"
        case .skill: return "wand.and.stars"
        case .file: return "doc"
		case .command: return "terminal"
        }
    }
}

private struct PendingChatAttachment: Identifiable {
    let id = UUID()
    let data: Data
    let filename: String
    let contentType: String
    let kind: String
}

private func installedMacAppNames() -> [String] {
    let applicationDirectories = [
        URL(fileURLWithPath: "/Applications", isDirectory: true),
        URL(fileURLWithPath: "/System/Applications", isDirectory: true),
        FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("Applications", isDirectory: true),
    ]
    var names = Set<String>()
    for directory in applicationDirectories {
        guard let enumerator = FileManager.default.enumerator(
            at: directory,
            includingPropertiesForKeys: [.isDirectoryKey],
            options: [.skipsHiddenFiles, .skipsPackageDescendants]
        ) else { continue }
        for case let url as URL in enumerator where url.pathExtension == "app" {
            names.insert(url.deletingPathExtension().lastPathComponent)
            enumerator.skipDescendants()
        }
    }
    return names.sorted { $0.localizedCaseInsensitiveCompare($1) == .orderedAscending }
}

private func desktopAppCommands() -> [ChatComposerSuggestion] {
    installedMacAppNames().map { name in
        let displayName = name == "Music" ? "Apple Music" : name
        return ChatComposerSuggestion(
            id: "app.\(name.lowercased())",
            kind: .app,
            title: displayName,
            subtitle: name == "Music" ? "Open or create a playlist" : "Open this Mac app",
            insertion: "@\(displayName) "
        )
    }
}

private func nativeAppCommandInstruction(for content: String) -> String {
    let trimmed = content.trimmingCharacters(in: .whitespacesAndNewlines)
    guard trimmed.hasPrefix("@") else { return content }
    let unprefixed = String(trimmed.dropFirst())
    let displayNames = installedMacAppNames().flatMap { $0 == "Music" ? [$0, "Apple Music"] : [$0] }
    guard let matchedName = displayNames.sorted(by: { $0.count > $1.count }).first(where: {
        unprefixed.lowercased() == $0.lowercased() || unprefixed.lowercased().hasPrefix($0.lowercased() + " ")
    }) else { return content }
    let appName = matchedName == "Apple Music" ? "Music" : matchedName

    let command = String(unprefixed.dropFirst(matchedName.count)).trimmingCharacters(in: .whitespacesAndNewlines)
    let lowered = command.lowercased()
    if appName == "Music", (lowered.hasPrefix("create") || lowered.hasPrefix("make")), let playlistRange = lowered.range(of: "playlist") {
        var playlistName = String(command[playlistRange.upperBound...]).trimmingCharacters(in: .whitespacesAndNewlines)
        for namePrefix in ["named ", "called "] where playlistName.lowercased().hasPrefix(namePrefix) {
            playlistName = String(playlistName.dropFirst(namePrefix.count)).trimmingCharacters(in: .whitespacesAndNewlines)
        }
        playlistName = playlistName.trimmingCharacters(in: CharacterSet(charactersIn: "\\\"' "))
        if !playlistName.isEmpty {
            return """
            The user invoked the Music desktop app command. Create the playlist named \"\(playlistName)\" by calling mac_control with action=create_music_playlist and playlist_name=\"\(playlistName)\". Do not use run_applescript. If playlist permission is disabled, explain how to enable Allow Apple Music playlists in Desktop Settings.
            """
        }
    }

    let request = command.isEmpty ? "Open the app." : command
    return """
    The user invoked the Mac desktop app command @\(appName). Call mac_control with action=open_app, app=\"\(appName)\", and background=false. Then help with this request in that app: \(request). If opening apps is disabled, explain how to enable Allow opening Mac apps in Desktop Settings.
    """
}

struct ChatView: View {
    @ObservedObject private var chat: NativeChatStore
    @StateObject private var discussionProviders = NativeProviderStore()
    private let port: Int
    @AppStorage("launcher.design.theme") private var savedTheme = LauncherTheme.light.rawValue
    @AppStorage("launcher.design.accent") private var savedAccent = LauncherAccent.theme.rawValue
    @AppStorage("launcher.design.density") private var savedDensity = ChatDensity.comfortable.rawValue
    @AppStorage("launcher.design.surface") private var savedSurface = MessageSurface.cards.rawValue
    @AppStorage("launcher.design.fontScale") private var fontScale = 1.0
    @AppStorage("launcher.design.backgroundPath") private var backgroundPath = ""
    @Environment(\.colorScheme) private var systemColorScheme
    @State private var isRecording = false
    @State private var isTranscribingVoice = false
    @State private var thinkingButtonGlow = false
    @State private var recorder: AVAudioRecorder?
    @State private var recordingURL: URL?
    @State private var suggestions: [ChatComposerSuggestion] = []
    @State private var appCommands: [ChatComposerSuggestion] = []
    @State private var pendingAttachment: PendingChatAttachment?
    @State private var isFolderDropTargeted = false
    @State private var discussionModelOverride = ""

    init(port: Int, chat: NativeChatStore) {
        self.port = port
        _chat = ObservedObject(wrappedValue: chat)
    }

    private var theme: LauncherTheme {
        launcherThemePreference(from: savedTheme).resolved(for: systemColorScheme)
    }
    private var accent: Color { (LauncherAccent(rawValue: savedAccent) ?? .theme).color ?? theme.accent }
    private var density: ChatDensity { ChatDensity(rawValue: savedDensity) ?? .comfortable }
    private var messageSurface: MessageSurface { MessageSurface(rawValue: savedSurface) ?? .cards }
    private var backgroundImage: NSImage? {
        if !backgroundPath.isEmpty {
            return NSImage(contentsOf: URL(fileURLWithPath: backgroundPath))
        }
        guard let bundledURL = Bundle.main.url(forResource: "creation-of-adam", withExtension: "jpg") else {
            return nil
        }
        return NSImage(contentsOf: bundledURL)
    }
    private var isConnectingToJame: Bool {
        chat.status != "Ready" && chat.messages.isEmpty && chat.lastError == nil
    }
    private var discussionProviderOptions: [(label: String, value: String)] {
        var options: [(label: String, value: String)] = [("Auto failover", "")]
        if let primary = discussionProviders.models.first(where: {
            $0.modelName == discussionProviders.selectedModel
        }) {
            options.append(("Use \(discussionProviders.providerName(for: primary))", primary.modelName))
        }
        if let fallback = discussionProviders.models.first(where: {
            $0.modelName == discussionProviders.selectedFallbackModel
        }) {
            options.append(("Use \(discussionProviders.providerName(for: fallback))", fallback.modelName))
        }
        return options
    }
    private var composerBackground: Color {
        theme == .light ? Color.white.opacity(0.98) : theme.panel.opacity(0.98)
    }
    private var composerSurface: Color {
        theme == .light ? JameBrand.ink.opacity(0.055) : Color.white.opacity(0.07)
    }
    private var composerBorder: Color {
        theme == .light ? JameBrand.ink.opacity(0.24) : Color.white.opacity(0.18)
    }

    var body: some View {
        VStack(spacing: 0) {
            if chat.showsTaskFiles {
            Button {
                chooseWorkspace()
            } label: {
                HStack(spacing: 8) {
                    Image(systemName: "folder.fill")
                        .foregroundStyle(accent)
                    VStack(alignment: .leading, spacing: 1) {
                        Text("Task files")
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(.secondary)
                        Text(chat.workspaceName)
                            .font(.subheadline.weight(.medium))
                            .lineLimit(1)
                        Text(chat.workspacePath)
                            .font(.caption2.monospaced())
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                            .truncationMode(.middle)
                    }
                    Spacer()
                    Image(systemName: "chevron.right")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                }
                .padding(.horizontal, 18)
                .padding(.vertical, 9)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .help("Choose agent workspace")
            .background(theme.panel.opacity(0.9))
            }
			NativeAgentControlBar(
				chat: chat,
				accent: accent,
				fontScale: fontScale,
				modelName: discussionProviders.selectedModel,
				providerName: discussionProviders.providerNames[discussionProviders.selectedModel] ?? ""
			)
			NativeActivityRail(activity: chat.activity, accent: accent, fontScale: fontScale)
			if let error = chat.lastError {
                HStack(alignment: .top, spacing: 10) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundStyle(.orange)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(error.title).font(.subheadline.weight(.semibold))
                        Text(error.detail).font(.caption).foregroundStyle(.secondary)
                    }
                    Spacer(minLength: 8)
                    if chat.canRetryLastMessage {
                        Button("Retry message") { chat.retryLastMessage() }
                            .buttonStyle(.plain)
                            .foregroundStyle(Color.white)
                            .padding(.horizontal, 10)
                            .frame(height: 28)
                            .background(JameBrand.orange)
                            .overlay(Rectangle().stroke(JameBrand.orange.opacity(0.85)))
                    } else {
                        Button("Retry connection") { chat.retryConnection() }
                            .buttonStyle(.plain)
                            .padding(.horizontal, 10)
                            .frame(height: 28)
                            .background(composerSurface)
                            .overlay(Rectangle().stroke(composerBorder))
                    }
                    if chat.canContinueConversation {
                        Button("Continue") { chat.continueConversation(modelOverride: discussionModelOverride) }
                            .buttonStyle(.plain)
                            .padding(.horizontal, 10)
                            .frame(height: 28)
                            .background(composerSurface)
                            .overlay(Rectangle().stroke(composerBorder))
                    }
                    Button { chat.dismissError() } label: {
                        Image(systemName: "xmark")
                    }
                    .buttonStyle(.borderless)
                    .help("Dismiss error")
                }
                .padding(.horizontal, 18)
                .padding(.vertical, 10)
                .background(Color.orange.opacity(theme == .light ? 0.12 : 0.18))
            }
            ScrollViewReader { proxy in
                HStack(spacing: 0) {
                    ScrollView {
                        LazyVStack(alignment: .leading, spacing: density.messageSpacing) {
                        ForEach(chat.messages) { message in
                            Group {
                                if message.role == "memory" {
                                    NativeMemoryChangedCard(
                                        summary: message.content,
                                        accent: accent,
                                        fontScale: fontScale
                                    )
                                } else {
                                    VStack(alignment: .leading, spacing: 6) {
                                        Text(message.role == "user" ? "you >" : message.role == "error" ? "error >" : "jame >")
                                            .font(.system(size: 10 * fontScale, weight: .semibold, design: .monospaced))
                                            .foregroundStyle(message.role == "user" ? accent : message.role == "error" ? .red : JameBrand.orangeSoft)
                                        if let planSteps = nativePlanSteps(from: message.content), message.role == "assistant" {
                                            NativePlanCard(steps: planSteps, accent: accent, fontScale: fontScale)
                                        } else {
                                            Text(message.content).textSelection(.enabled)
                                        }
                                    }
                                    .font(.system(size: 14 * fontScale, design: .monospaced))
                                    .foregroundStyle(theme.text)
                                    .padding(density.messagePadding)
                                    .frame(maxWidth: message.role == "user" ? 520 : .infinity, alignment: .leading)
                                    .background(messageSurface == .cards ? (message.role == "user" ? accent.opacity(theme == .light ? 0.14 : 0.22) : message.role == "error" ? Color.red.opacity(0.18) : Color.white.opacity(theme == .light ? 0.82 : 0.06)) : .clear)
                                    .clipShape(Rectangle())
                                }
                            }
                                .frame(maxWidth: .infinity, alignment: message.role == "user" ? .trailing : .leading)
                                .id(message.id)
                        }
                        if chat.isResponseInProgress {
                            Text("Thinking... 💭")
                                .font(.system(size: 12 * fontScale, weight: .semibold, design: .monospaced))
                                .foregroundStyle(JameBrand.orange)
                        }
                        }.padding(density.contentPadding)
                    }
                    .frame(maxWidth: .infinity)
                }
                .background(chatBackground)
                .onChange(of: chat.messages.count) { _, _ in if let last = chat.messages.last { proxy.scrollTo(last.id, anchor: .bottom) } }
            }
            Divider().overlay(composerBorder)
            if let attachment = pendingAttachment {
                HStack(spacing: 8) {
                    Image(systemName: attachment.kind == "image" ? "photo" : "paperclip")
                        .foregroundStyle(accent)
                    Text(attachment.filename)
                        .lineLimit(1)
                        .font(.caption)
                    Text("Add a message, then send when ready")
                        .lineLimit(1)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Spacer()
                    Button {
                        pendingAttachment = nil
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                    }
                    .buttonStyle(.borderless)
                    .help("Remove attachment")
                }
                .padding(.horizontal, 14)
                .padding(.vertical, 8)
                .background(composerBackground)
            }
            if chat.showsTaskFiles {
            HStack(spacing: 7) {
                Image(systemName: isFolderDropTargeted ? "folder.badge.plus" : "folder.fill")
                    .foregroundStyle(isFolderDropTargeted ? JameBrand.orange : accent)
                Text(isFolderDropTargeted ? "Release to organize Jame's task files here" : "Drop Desktop, Documents, Downloads, or a project folder here")
                    .font(.caption.weight(isFolderDropTargeted ? .semibold : .regular))
                    .foregroundStyle(isFolderDropTargeted ? JameBrand.orange : .secondary)
                Spacer()
                Text(chat.workspacePath)
                    .font(.caption2.monospaced())
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                    .truncationMode(.middle)
                    .frame(maxWidth: 320, alignment: .trailing)
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 7)
            .background(composerBackground)
            .overlay(alignment: .top) {
                Rectangle()
                    .fill(composerBorder)
                    .frame(height: 1)
                    .allowsHitTesting(false)
            }
            .fixedSize(horizontal: false, vertical: true)
            }
            ZStack(alignment: .bottomLeading) {
                HStack(alignment: .bottom) {
                    Button {
                        chooseWorkspace()
                    } label: {
                        Image(systemName: "folder")
                    }
                    .buttonStyle(.plain)
                    .frame(width: 34, height: 34)
                    .background(composerSurface)
                    .overlay(Rectangle().stroke(composerBorder))
                    .help("Choose agent workspace")
                    .disabled(chat.isResponseInProgress)

                    Button {
                        uploadItem()
                    } label: {
                        Image(systemName: "paperclip")
                    }
                    .buttonStyle(.plain)
                    .frame(width: 34, height: 34)
                    .background(composerSurface)
                    .overlay(Rectangle().stroke(composerBorder))
                    .help("Upload a file or workspace skill")
                    .disabled(chat.isResponseInProgress)

                    Button {
                        toggleRecording()
                    } label: {
                        Image(systemName: isRecording ? "stop.circle.fill" : isTranscribingVoice ? "waveform.badge.magnifyingglass" : "mic.fill")
                            .foregroundStyle(isRecording ? Color.red : accent)
                    }
                    .buttonStyle(.plain)
                    .frame(width: 34, height: 34)
                    .background(composerSurface)
                    .overlay(Rectangle().stroke(composerBorder))
                    .help(isRecording ? "Stop and transcribe recording" : isTranscribingVoice ? "Transcribing voice into the message box" : "Record a voice message")
                    .disabled((chat.isResponseInProgress || isTranscribingVoice) && !isRecording)

                    TextField("type a message…", text: $chat.draft, axis: .vertical)
                        .font(.system(size: 14 * fontScale, design: .monospaced))
                        .lineLimit(1...5)
                        .textFieldStyle(.plain)
                        .padding(.horizontal, 11)
                        .padding(.vertical, 8)
                        .background(composerSurface)
                        .overlay {
                            Rectangle()
                                .stroke(composerBorder, lineWidth: theme == .light ? 1.2 : 1)
                        }
                        .onChange(of: chat.draft) { _, value in updateSuggestions(for: value) }
                    Button { sendComposer() } label: {
                        Text(chat.isResponseInProgress ? "Queue" : "Send")
                            .font(.system(size: 13, weight: .semibold, design: .rounded))
                            .frame(minWidth: 36)
                    }
                        .buttonStyle(.plain)
                        .foregroundStyle(Color.white)
                        .padding(.horizontal, 16)
                        .frame(height: 34)
                        .background(chat.isResponseInProgress ? accent.opacity(0.78) : accent)
                        .overlay(Rectangle().stroke(accent.opacity(chat.isResponseInProgress ? 1 : 0.85), lineWidth: chat.isResponseInProgress ? 1.5 : 1))
                        .shadow(
                            color: chat.isResponseInProgress ? accent.opacity(thinkingButtonGlow ? 0.92 : 0.28) : .clear,
                            radius: chat.isResponseInProgress ? (thinkingButtonGlow ? 13 : 4) : 0
                        )
                        .scaleEffect(chat.isResponseInProgress && thinkingButtonGlow ? 1.035 : 1)
                        .animation(
                            chat.isResponseInProgress
                                ? .easeInOut(duration: 0.68).repeatForever(autoreverses: true)
                                : .easeOut(duration: 0.18),
                            value: thinkingButtonGlow
                        )
                        .onAppear { thinkingButtonGlow = chat.isResponseInProgress }
                        .onChange(of: chat.isResponseInProgress) { _, isInProgress in
                            thinkingButtonGlow = isInProgress
                        }
                        .disabled(chat.draft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                        .accessibilityLabel(chat.isResponseInProgress ? "Queue message" : "Send")
                        .help(chat.isResponseInProgress ? "Queue this message for Jame" : "Send message")
                        .keyboardShortcut(.defaultAction)
                    if discussionProviders.selectedFallbackModel.isEmpty == false {
                        JameDropdownPicker(
                            label: "Discussion provider",
                            options: discussionProviderOptions,
                            selection: $discussionModelOverride,
                            minWidth: 168
                        )
                        .help("Choose the provider for this discussion. Auto failover uses the global primary and fallback pair.")
                    }
                }
                .padding(14)

                if !suggestions.isEmpty {
                    ScrollView {
                        LazyVStack(alignment: .leading, spacing: 3) {
                            ForEach(suggestions) { suggestion in
                                Button {
                                    applySuggestion(suggestion)
                                } label: {
                                    HStack(spacing: 8) {
                                        Image(systemName: suggestion.icon).frame(width: 16)
                                        Text(suggestion.title).font(.subheadline.weight(.medium))
                                        Text(suggestion.subtitle).font(.caption).foregroundStyle(.secondary).lineLimit(1)
                                        Spacer()
                                    }
                                    .padding(.horizontal, 12).padding(.vertical, 6)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    .frame(width: 380, height: min(CGFloat(suggestions.count) * 38 + 10, 220))
                    .background(composerBackground)
                    .clipShape(Rectangle())
                    .overlay(Rectangle().stroke(composerBorder))
                    .shadow(color: .black.opacity(0.24), radius: 10, y: 4)
                    .offset(x: 56, y: -70)
                    .zIndex(1)
                }
            }
            .background(composerBackground)
            .contentShape(Rectangle())
            .overlay {
                Rectangle()
                    .stroke(
                        isFolderDropTargeted ? JameBrand.orange : composerBorder,
                        style: StrokeStyle(
                            lineWidth: isFolderDropTargeted ? 2 : 1,
                            dash: isFolderDropTargeted ? [7, 5] : []
                        )
                    )
                    .padding(isFolderDropTargeted ? 4 : 0)
                    .allowsHitTesting(false)
            }
            .shadow(
                color: Color.black.opacity(theme == .light ? 0.16 : 0.28),
                radius: 10,
                y: -2
            )
            .onDrop(
                of: [UTType.fileURL.identifier],
                isTargeted: $isFolderDropTargeted,
                perform: handleComposerDrop(providers:)
            )
        }
        .background(chatBackground)
        // Chat's canvas and composer have a large natural fitting height.
        // Keep that ideal size compressible so entering Chat does not enlarge
        // the user's existing desktop window.
        .frame(
            minWidth: 0,
            idealWidth: 680,
            maxWidth: .infinity,
            minHeight: 0,
            idealHeight: 480,
            maxHeight: .infinity
        )
        .preferredColorScheme(launcherThemePreference(from: savedTheme).preferredColorScheme)
        .overlay {
            if isConnectingToJame {
                jameLoadingScreen
            }
        }
        .task {
            appCommands = desktopAppCommands()
            chat.startGatewayAndConnect()
            await discussionProviders.load(port: port)
        }
        .onReceive(NotificationCenter.default.publisher(for: .jameclawNewChat)) { _ in
            chat.startNewChat()
        }
        .onReceive(NotificationCenter.default.publisher(for: .jameclawResumeSession)) { notification in
            guard let request = notification.object as? NativeSessionResumeRequest else { return }
            chat.resumeSession(request)
        }
        .onReceive(NotificationCenter.default.publisher(for: .jameclawWorkspaceChanged)) { notification in
            guard let workspacePath = notification.userInfo?["workspacePath"] as? String else { return }
            chat.reflectWorkspace(URL(fileURLWithPath: workspacePath, isDirectory: true))
            guard notification.userInfo?["isDirectory"] as? Bool == false,
                  let selectedPath = notification.userInfo?["selectedPath"] as? String else { return }
            let instruction = "Work with this document: \(selectedPath)"
            chat.draft = chat.draft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                ? instruction
                : instruction + "\n" + chat.draft
        }
    }

    private var jameLoadingScreen: some View {
        ZStack {
            JameBrand.ink.ignoresSafeArea()
            RadialGradient(
                colors: [JameBrand.orange.opacity(0.18), .clear],
                center: .center,
                startRadius: 10,
                endRadius: 320
            )
            .ignoresSafeArea()
            VStack(spacing: 16) {
                HStack(spacing: 0) {
                    Text("Jame")
                        .foregroundStyle(.white)
                    Text(".")
                        .foregroundStyle(Color.orange)
                }
                .font(.system(size: 34 * fontScale, weight: .semibold, design: .rounded))
                ProgressView()
                    .controlSize(.large)
                    .tint(accent)
                Text(chat.status)
                    .font(.system(size: 14 * fontScale, weight: .medium, design: .rounded))
                    .foregroundStyle(.white.opacity(0.9))
                Text("Starting your private local gateway on localhost…")
                    .font(.system(size: 13 * fontScale, design: .monospaced))
                    .foregroundStyle(.secondary)
            }
            .padding(42)
        }
    }

    private func updateSuggestions(for input: String) {
        guard let match = input.range(of: "(?:^|\\s)([@/])[^\\s]*$", options: .regularExpression) else {
            suggestions = []
            return
        }
        let token = String(input[match])
        guard let trigger = token.first(where: { $0 == "@" || $0 == "/" }) else {
            suggestions = []
            return
        }
        let query = String(token.drop { $0 == " " || $0 == trigger }).lowercased()
        let matchingAppItems = trigger == "@" ? appCommands.filter {
            query.isEmpty || $0.title.lowercased().contains(query) || $0.subtitle.lowercased().contains(query)
        } : []
        let builtInCommands: [ChatComposerSuggestion] = [
            ("status", "Show agent and task status"), ("stop", "Stop the active task"),
            ("queue", "Show or clear queued messages"), ("model", "Show or switch model"),
            ("new", "Start a fresh conversation"), ("sessions", "Open saved sessions"),
            ("stats", "Show session context"), ("usage", "Show context usage"),
            ("insights", "Summarize conversation state"), ("undo", "Undo the last turn"),
            ("compact", "Compact older context"), ("automations", "List automations"),
            ("run", "Run an automation"), ("pause", "Pause an automation"),
            ("resume", "Resume an automation"), ("approvals", "Show safety controls"), ("help", "List commands")
        ]
        .filter { query.isEmpty || $0.0.contains(query) || $0.1.lowercased().contains(query) }
        .map { name, description in
            ChatComposerSuggestion(id: "command.\(name)", kind: .command, title: "/\(name)", subtitle: description, insertion: "/\(name) ")
        }
        if trigger == "@" {
            suggestions = matchingAppItems
        }
        Task {
            do {
                let skillsData = try await URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: "/api/skills")).0
                let skills = try JSONDecoder().decode(NativeSkillsResponse.self, from: skillsData).skills
                var items = skills
                    .filter { query.isEmpty || $0.name.lowercased().contains(query) || $0.description.lowercased().contains(query) }
                    .prefix(8)
                    .map { skill in
                        ChatComposerSuggestion(id: skill.id, kind: .skill, title: skill.name, subtitle: skill.description.isEmpty ? "\(skill.source) skill" : skill.description, insertion: trigger == "/" ? "/\(skill.name) " : "@skill:\(skill.name) ")
                    }
                if trigger == "/" {
                    items.insert(contentsOf: builtInCommands, at: 0)
                }
                if trigger == "@" {
                    items.insert(contentsOf: matchingAppItems, at: 0)
                    let escaped = query.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? ""
                    let fileData = try await URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: "/api/files/search", queryItems: [URLQueryItem(name: "q", value: escaped), URLQueryItem(name: "limit", value: "8")])).0
                    let files = try JSONDecoder().decode(NativeFileSearchResponse.self, from: fileData).items
                    items.append(contentsOf: files.map { file in
                        ChatComposerSuggestion(id: file.id, kind: .file, title: file.name, subtitle: file.directory, insertion: "@\"\(file.path)\" ")
                    })
                }
                guard input == chat.draft else { return }
                suggestions = items
            } catch {
                guard input == chat.draft else { return }
                suggestions = matchingAppItems
            }
        }
    }

    private func applySuggestion(_ suggestion: ChatComposerSuggestion) {
        guard let range = chat.draft.range(of: "(?:^|\\s)[@/][^\\s]*$", options: .regularExpression) else { return }
        let prefix = String(chat.draft[..<range.lowerBound])
        let leadingSpace = chat.draft[range].first?.isWhitespace == true ? " " : ""
        chat.draft = prefix + leadingSpace + suggestion.insertion
        suggestions = []
    }

    @ViewBuilder
    private var chatBackground: some View {
        ZStack {
            theme.background
            if let image = backgroundImage {
                GeometryReader { geometry in
                    Image(nsImage: image)
                        .resizable()
                        .scaledToFill()
                        .frame(width: geometry.size.width, height: geometry.size.height)
                        .clipped()
                }
                .opacity(theme == .light ? 0.28 : 0.20)
            }
        }
    }

    private func uploadFile(_ url: URL) {
        let didAccess = url.startAccessingSecurityScopedResource()
        defer { if didAccess { url.stopAccessingSecurityScopedResource() } }
        do {
            let data = try Data(contentsOf: url)
            let type = UTType(filenameExtension: url.pathExtension)
            let contentType = type?.preferredMIMEType ?? "application/octet-stream"
            let kind = type?.conforms(to: .image) == true ? "image" : "file"
            pendingAttachment = PendingChatAttachment(
                data: data,
                filename: url.lastPathComponent,
                contentType: contentType,
                kind: kind
            )
        } catch {
            chat.status = "Could not read that file."
        }
    }

    private func handleComposerDrop(providers: [NSItemProvider]) -> Bool {
        guard let provider = providers.first(where: {
            $0.hasItemConformingToTypeIdentifier(UTType.fileURL.identifier)
        }) else {
            chat.status = "Drop a folder from Finder here."
            return false
        }

        provider.loadItem(forTypeIdentifier: UTType.fileURL.identifier, options: nil) { item, error in
            let droppedURL: URL?
            if let url = item as? URL {
                droppedURL = url
            } else if let url = item as? NSURL {
                droppedURL = url as URL
            } else if let data = item as? Data {
                droppedURL = URL(dataRepresentation: data, relativeTo: nil)
            } else if let rawValue = item as? String {
                droppedURL = rawValue.hasPrefix("file:")
                    ? URL(string: rawValue)
                    : URL(fileURLWithPath: rawValue)
            } else {
                droppedURL = nil
            }

            Task { @MainActor in
                guard error == nil, let droppedURL else {
                    chat.status = "Could not read the dropped item."
                    return
                }
                useDroppedItem(droppedURL)
            }
        }
        return true
    }

    @MainActor
    private func useDroppedItem(_ droppedURL: URL) {
        let didAccess = droppedURL.startAccessingSecurityScopedResource()
        defer { if didAccess { droppedURL.stopAccessingSecurityScopedResource() } }

        let resolvedURL = droppedURL.standardizedFileURL.resolvingSymlinksInPath()
        var isDirectory: ObjCBool = false
        guard FileManager.default.fileExists(atPath: resolvedURL.path, isDirectory: &isDirectory) else {
            chat.status = "That dropped item is no longer available."
            return
        }
        guard approveDocumentAccess(
            to: resolvedURL,
            action: isDirectory.boolValue ? "use this folder for organized task files" : "attach and read this document"
        ) else {
            chat.status = "Document access was not approved."
            return
        }

        if isDirectory.boolValue {
            chat.setWorkspace(resolvedURL)
        } else {
            uploadFile(resolvedURL)
            chat.status = "Attached \(resolvedURL.lastPathComponent)."
        }
    }

    private func sendComposer() {
        let content = chat.draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !content.isEmpty || pendingAttachment != nil else { return }
        if pendingAttachment == nil, let query = localFileSearchQuery(from: content) {
            chat.draft = ""
            suggestions = []
            Task { await runFastFileSearch(query: query, originalRequest: content) }
            return
        }
        guard let attachment = pendingAttachment else {
            chat.send(modelOverride: discussionModelOverride)
            return
        }

        chat.draft = ""
        pendingAttachment = nil
        chat.sendMedia(
            data: attachment.data,
            filename: attachment.filename,
            contentType: attachment.contentType,
            kind: attachment.kind,
            content: content,
            modelOverride: discussionModelOverride
        )
    }

    private func runFastFileSearch(query: String, originalRequest: String) async {
        do {
            let data = try await URLSession.shared.data(
                from: authenticatedConsoleURL(
                    port: port,
                    path: "/api/files/search",
                    queryItems: [URLQueryItem(name: "q", value: query), URLQueryItem(name: "limit", value: "12")]
                )
            ).0
            let results = try JSONDecoder().decode(NativeFileSearchResponse.self, from: data).items
            let response: String
            if results.isEmpty {
                response = "I couldn't find files matching \"\(query)\" in JameClaw's allowed local folders."
            } else {
                let lines = results.map { "• \($0.name) — \($0.path)" }.joined(separator: "\n")
                response = "Found \(results.count) local file\(results.count == 1 ? "" : "s") for \"\(query)\":\n\(lines)"
            }
            chat.addFastPathExchange(user: originalRequest, response: response)
        } catch {
            chat.addFastPathExchange(user: originalRequest, response: "Local file search is not available right now. JameClaw can still search through the normal agent workflow.")
        }
    }

    private func localFileSearchQuery(from text: String) -> String? {
        let normalized = text.trimmingCharacters(in: .whitespacesAndNewlines)
        let prefixes = [
            "search my pc for ", "search my computer for ", "search files for ",
            "find file ", "find files ", "find on my pc ", "find on my computer ",
        ]
        let lower = normalized.lowercased()
        for prefix in prefixes where lower.hasPrefix(prefix) {
            let query = String(normalized.dropFirst(prefix.count)).trimmingCharacters(in: .whitespacesAndNewlines)
            if !query.isEmpty { return query }
        }
        return nil
    }

    private func uploadItem() {
        let panel = NSOpenPanel()
        panel.title = "Upload File or JameClaw Skill"
        panel.message = "Choose a file to attach, or choose SKILL.md / a skill folder to import into this workspace."
        panel.prompt = "Upload"
        panel.canChooseFiles = true
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false

        guard panel.runModal() == .OK, let sourceURL = panel.url else { return }
        let didAccess = sourceURL.startAccessingSecurityScopedResource()
        defer { if didAccess { sourceURL.stopAccessingSecurityScopedResource() } }

        var isDirectory: ObjCBool = false
        guard FileManager.default.fileExists(atPath: sourceURL.path, isDirectory: &isDirectory) else {
            chat.status = "Could not find that item."
            return
        }
        guard approveDocumentAccess(
            to: sourceURL,
            action: isDirectory.boolValue ? "read and import this folder" : "attach and read this document"
        ) else {
            chat.status = "Document access was not approved."
            return
        }
        if isDirectory.boolValue || sourceURL.lastPathComponent.caseInsensitiveCompare("SKILL.md") == .orderedSame {
            importSkill(sourceURL)
        } else {
            uploadFile(sourceURL)
        }
    }

    private func importSkill(_ sourceURL: URL) {
        do {
            let fileManager = FileManager.default
            let sourceSkillFile: URL
            let sourceDirectory: URL
            let sourceIsDirectory: Bool

            var isDirectory: ObjCBool = false
            guard fileManager.fileExists(atPath: sourceURL.path, isDirectory: &isDirectory) else {
                throw SkillUploadError.missingSkillFile
            }
            sourceIsDirectory = isDirectory.boolValue
            if sourceIsDirectory {
                sourceDirectory = sourceURL
                sourceSkillFile = sourceURL.appendingPathComponent("SKILL.md")
            } else {
                guard sourceURL.lastPathComponent.caseInsensitiveCompare("SKILL.md") == .orderedSame else {
                    throw SkillUploadError.chooseSkillFile
                }
                sourceSkillFile = sourceURL
                sourceDirectory = sourceURL.deletingLastPathComponent()
            }

            guard fileManager.fileExists(atPath: sourceSkillFile.path) else {
                throw SkillUploadError.missingSkillFile
            }
            let skillDocument = try String(contentsOf: sourceSkillFile, encoding: .utf8)
            let skillName = skillName(from: skillDocument, fallback: sourceDirectory.lastPathComponent)
            let skillsDirectory = jameWorkspaceURL().appendingPathComponent("skills", isDirectory: true)
            let destinationDirectory = skillsDirectory.appendingPathComponent(skillName, isDirectory: true)

            guard !fileManager.fileExists(atPath: destinationDirectory.path) else {
                throw SkillUploadError.alreadyExists(skillName)
            }
            try fileManager.createDirectory(at: skillsDirectory, withIntermediateDirectories: true)
            if sourceIsDirectory {
                try fileManager.copyItem(at: sourceDirectory, to: destinationDirectory)
            } else {
                try fileManager.createDirectory(at: destinationDirectory, withIntermediateDirectories: true)
                try fileManager.copyItem(at: sourceSkillFile, to: destinationDirectory.appendingPathComponent("SKILL.md"))
            }
            chat.status = "Imported skill: \(skillName)"
            chat.sendSkillImported(skillName)
        } catch {
            chat.status = error.localizedDescription
        }
    }

    private func skillName(from document: String, fallback: String) -> String {
        let declaredName = document
            .split(whereSeparator: \.isNewline)
            .first { $0.trimmingCharacters(in: .whitespaces).hasPrefix("name:") }
            .map { String($0).split(separator: ":", maxSplits: 1)[1].trimmingCharacters(in: .whitespacesAndNewlines) }
        let candidate = declaredName?.isEmpty == false ? declaredName! : fallback
        let slug = candidate.lowercased()
            .replacingOccurrences(of: "[^a-z0-9]+", with: "-", options: .regularExpression)
            .trimmingCharacters(in: CharacterSet(charactersIn: "-"))
        return slug.isEmpty || slug == "skill" ? "imported-skill-\(UUID().uuidString.prefix(8))" : slug
    }

    private func chooseWorkspace() {
        let panel = NSOpenPanel()
        panel.title = "Choose JameClaw Task Folder"
        panel.message = "Choose where Jame should organize task files. Desktop, Documents, and Downloads use a dedicated JameClaw subfolder."
        panel.prompt = "Use Task Folder"
        panel.canChooseFiles = false
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false
        panel.canCreateDirectories = true

        guard panel.runModal() == .OK, let workspaceURL = panel.url else { return }
        guard approveDocumentAccess(to: workspaceURL, action: "use this folder for organized task files") else { return }
        chat.setWorkspace(workspaceURL)
    }

    private func toggleRecording() {
        if isRecording {
            recorder?.stop()
            isRecording = false
            defer { recorder = nil }
            guard let url = recordingURL else { return }
            recordingURL = nil
            Task { await transcribeRecording(at: url) }
            return
        }

        AVCaptureDevice.requestAccess(for: .audio) { granted in
            DispatchQueue.main.async {
                guard granted else {
                    chat.status = "Microphone access is required to record a voice message."
                    return
                }
                let url = FileManager.default.temporaryDirectory.appendingPathComponent("jame-voice-\(UUID().uuidString).m4a")
                let settings: [String: Any] = [
                    AVFormatIDKey: Int(kAudioFormatMPEG4AAC),
                    AVSampleRateKey: 44_100,
                    AVNumberOfChannelsKey: 1,
                    AVEncoderAudioQualityKey: AVAudioQuality.high.rawValue,
                ]
                do {
                    let newRecorder = try AVAudioRecorder(url: url, settings: settings)
                    newRecorder.record()
                    recorder = newRecorder
                    recordingURL = url
                    isRecording = true
                } catch {
                    chat.status = "Could not start recording."
                }
            }
        }
    }

    @MainActor
    private func transcribeRecording(at url: URL) async {
        isTranscribingVoice = true
        chat.status = "Transcribing voice into the message box…"
        defer {
            isTranscribingVoice = false
            try? FileManager.default.removeItem(at: url)
        }

        do {
            let audio = try Data(contentsOf: url)
            let boundary = "JameClawVoice-\(UUID().uuidString)"
            var body = Data()
            body.append("--\(boundary)\r\n".data(using: .utf8)!)
            body.append("Content-Disposition: form-data; name=\"audio\"; filename=\"recording.m4a\"\r\n".data(using: .utf8)!)
            body.append("Content-Type: audio/mp4\r\n\r\n".data(using: .utf8)!)
            body.append(audio)
            body.append("\r\n--\(boundary)--\r\n".data(using: .utf8)!)

            var request = authenticatedConsoleRequest(port: port, path: "/api/voice/transcribe", method: "POST")
            request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
            request.httpBody = body
            request.timeoutInterval = 120

            let (data, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                let detail = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)
                throw NSError(
                    domain: "JameClawVoice",
                    code: (response as? HTTPURLResponse)?.statusCode ?? 1,
                    userInfo: [NSLocalizedDescriptionKey: detail?.isEmpty == false ? detail! : "Voice transcription failed."]
                )
            }
            struct VoiceTranscription: Decodable { let text: String }
            let transcription = try JSONDecoder().decode(VoiceTranscription.self, from: data)
            let text = transcription.text.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !text.isEmpty else {
                throw NSError(domain: "JameClawVoice", code: 2, userInfo: [NSLocalizedDescriptionKey: "The transcription was empty. Try recording again."])
            }
            let existing = chat.draft.trimmingCharacters(in: .whitespacesAndNewlines)
            chat.draft = existing.isEmpty ? text : existing + "\n" + text
            chat.status = "Voice transcription ready. Review it, then press Send."
        } catch {
            chat.status = error.localizedDescription.isEmpty ? "Could not transcribe the recording." : error.localizedDescription
        }
    }
}
