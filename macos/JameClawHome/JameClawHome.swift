import AppKit
import AVFoundation
import Foundation
import SwiftUI
import UniformTypeIdentifiers

private func authenticatedConsoleURL(port: Int, path: String = "/") -> URL {
    var components = URLComponents()
    components.scheme = "http"
    components.host = "localhost"
    components.port = port
    components.path = path
    let tokenURL = FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent(".jameclaw/launcher_access_token")
    if let token = try? String(contentsOf: tokenURL, encoding: .utf8).trimmingCharacters(in: .whitespacesAndNewlines), !token.isEmpty {
        components.queryItems = [URLQueryItem(name: "access_token", value: token)]
    }
    return components.url ?? URL(string: "http://localhost:\(port)")!
}

private func configuredLauncherPort() -> Int {
    let configURL = FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent(".jameclaw/launcher-config.json")
    guard let data = try? Data(contentsOf: configURL),
          let settings = try? JSONDecoder().decode(LauncherSettings.self, from: data) else {
        return 18800
    }
    return settings.port
}

final class HomeAppDelegate: NSObject, NSApplicationDelegate {
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        true
    }

    func applicationShouldTerminate(_ sender: NSApplication) -> NSApplication.TerminateReply {
        var request = URLRequest(url: authenticatedConsoleURL(port: configuredLauncherPort(), path: "/api/system/quit"))
        request.httpMethod = "POST"
        let completed = DispatchSemaphore(value: 0)
        URLSession.shared.dataTask(with: request) { _, _, _ in completed.signal() }.resume()
        _ = completed.wait(timeout: .now() + 0.6)
        return .terminateNow
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
    case terminal
    case midnight
    case light

    var id: String { rawValue }
    var label: String {
        switch self {
        case .terminal: return "Terminal"
        case .midnight: return "Midnight"
        case .light: return "Light"
        }
    }
    var colorScheme: ColorScheme { self == .light ? .light : .dark }
    var accent: Color {
        switch self {
        case .terminal: return Color(red: 0.95, green: 0.42, blue: 0.36)
        case .midnight: return Color(red: 0.36, green: 0.55, blue: 0.98)
        case .light: return Color(red: 0.72, green: 0.18, blue: 0.15)
        }
    }
    var background: Color {
        switch self {
        case .terminal: return Color(red: 0.045, green: 0.05, blue: 0.055)
        case .midnight: return Color(red: 0.035, green: 0.06, blue: 0.12)
        case .light: return Color(red: 0.96, green: 0.97, blue: 0.98)
        }
    }
    var panel: Color {
        switch self {
        case .terminal: return Color(red: 0.075, green: 0.08, blue: 0.09)
        case .midnight: return Color(red: 0.065, green: 0.10, blue: 0.19)
        case .light: return .white
        }
    }
    var text: Color { self == .light ? Color(red: 0.13, green: 0.15, blue: 0.18) : Color(red: 0.9, green: 0.92, blue: 0.88) }
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

@main
struct JameClawHomeApp: App {
    @NSApplicationDelegateAdaptor(HomeAppDelegate.self) private var appDelegate

    var body: some Scene {
        WindowGroup { JameRootView() }
            .windowResizability(.contentSize)
    }
}

struct JameRootView: View {
    @StateObject private var settings = LauncherSettingsStore()
    // The native launcher is a chat app first. Keep the supporting controls a
    // tab away so opening Jame from the menu-bar launcher always lands here.
    @State private var selectedTab = 0

    var body: some View {
        TabView(selection: $selectedTab) {
            ChatView(port: Int(settings.port) ?? 18800)
                .tabItem { Label("Chat", systemImage: "message.fill") }
                .tag(0)
            ArtifactsView()
                .tabItem { Label("Artifacts", systemImage: "shippingbox.fill") }
                .tag(1)
            SkillsView()
                .tabItem { Label("Skills", systemImage: "wand.and.stars") }
                .tag(2)
            QuickSettingsView(settings: settings)
                .tabItem { Label("Settings", systemImage: "gearshape") }
                .tag(3)
        }
        .frame(width: 820, height: 650)
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

private struct WorkspaceEntry: Identifiable {
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
                    Text(browser.title).font(.title3.weight(.bold))
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

struct ArtifactsView: View {
    @StateObject private var browser = WorkspaceBrowser(
        title: "Artifacts",
        directory: jameWorkspaceURL().appendingPathComponent("artifacts")
    )

    var body: some View {
        WorkspaceBrowserView(
            browser: browser,
            emptyTitle: "No artifacts yet",
            emptyDescription: "Files Jame creates in the workspace artifacts folder will appear here."
        )
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

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                VStack(alignment: .leading, spacing: 3) {
                    Text("Skills").font(.title3.weight(.bold))
                    Text(browser.directory.path).font(.caption.monospaced()).foregroundStyle(.secondary).lineLimit(1)
                }
                Spacer()
                Button { browser.refresh() } label: { Image(systemName: "arrow.clockwise") }
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
            if browser.entries.isEmpty {
                ContentUnavailableView(
                    "No workspace skills yet",
                    systemImage: "wand.and.stars",
                    description: Text("Add a skill here, or use the Web Console to manage skills.")
                )
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
        .sheet(isPresented: $showingAddSkill) {
            VStack(alignment: .leading, spacing: 16) {
                Text("Add a Skill").font(.title2.weight(.bold))
                Text("This creates a workspace skill that JameClaw loads from \(browser.directory.path).")
                    .foregroundStyle(.secondary)
                TextField("Skill name (for example, release-notes)", text: $skillName)
                TextField("What does this skill do?", text: $skillDescription, axis: .vertical)
                    .lineLimit(2...4)
                if !addError.isEmpty { Text(addError).font(.caption).foregroundStyle(.red) }
                HStack {
                    Spacer()
                    Button("Cancel") { showingAddSkill = false }
                    Button("Create Skill") { createSkill() }
                        .buttonStyle(.borderedProminent)
                        .disabled(skillName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
            .padding(24)
            .frame(width: 480)
        }
    }

    private func createSkill() {
        let displayName = skillName.trimmingCharacters(in: .whitespacesAndNewlines)
        let slug = displayName.lowercased()
            .replacingOccurrences(of: "[^a-z0-9]+", with: "-", options: .regularExpression)
            .trimmingCharacters(in: CharacterSet(charactersIn: "-"))
        guard !slug.isEmpty else {
            addError = "Use at least one letter or number in the skill name."
            return
        }
        let skillDirectory = browser.directory.appendingPathComponent(slug, isDirectory: true)
        let skillFile = skillDirectory.appendingPathComponent("SKILL.md")
        guard !FileManager.default.fileExists(atPath: skillDirectory.path) else {
            addError = "A skill named \(slug) already exists."
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
            try FileManager.default.createDirectory(at: skillDirectory, withIntermediateDirectories: true)
            try document.write(to: skillFile, atomically: true, encoding: .utf8)
            browser.refresh()
            showingAddSkill = false
            NSWorkspace.shared.activateFileViewerSelecting([skillFile])
        } catch {
            addError = "Could not create this skill."
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
            Text("Jame").font(.title2.weight(.bold))
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

struct QuickSettingsView: View {
    @ObservedObject var settings: LauncherSettingsStore
    @AppStorage("launcher.design.theme") private var savedTheme = LauncherTheme.terminal.rawValue
    @AppStorage("launcher.design.fontScale") private var fontScale = 1.0
    @AppStorage("launcher.design.backgroundPath") private var backgroundPath = ""
    @State private var showingBackgroundPicker = false

    var body: some View {
        Form {
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
            Section {
                Button("Save settings") { settings.save() }
                if !settings.saveStatus.isEmpty { Text(settings.saveStatus).font(.caption).foregroundStyle(.secondary) }
            }
            Section("Design") {
                Picker("Theme", selection: $savedTheme) {
                    ForEach(LauncherTheme.allCases) { theme in
                        Text(theme.label).tag(theme.rawValue)
                    }
                }
                Picker("Chat text size", selection: $fontScale) {
                    Text("Small").tag(0.88)
                    Text("Default").tag(1.0)
                    Text("Large").tag(1.15)
                    Text("Extra large").tag(1.3)
                }
                HStack {
                    Button("Choose chat background") { showingBackgroundPicker = true }
                    if !backgroundPath.isEmpty {
                        Button("Remove background", role: .destructive) { backgroundPath = "" }
                    }
                }
                Text("The selected image is stored locally and used behind the Chat view.")
                    .font(.caption).foregroundStyle(.secondary)
            }
        }
        .formStyle(.grouped)
        .padding(.top, 6)
        .fileImporter(
            isPresented: $showingBackgroundPicker,
            allowedContentTypes: [.image],
            allowsMultipleSelection: false
        ) { result in
            guard case let .success(urls) = result, let sourceURL = urls.first else { return }
            saveChatBackground(from: sourceURL)
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
}

struct NativeChatMessage: Identifiable {
    let id: String
    let role: String
    var content: String
}

private struct PendingNativeChatMessage {
    let id: String
    let content: String
}

@MainActor
final class NativeChatStore: ObservableObject {
    @Published var messages: [NativeChatMessage] = []
    @Published var draft = ""
    @Published var status = "Connecting…"
    @Published var isThinking = false

    private let port: Int
    private let sessionID = UUID().uuidString
    private var socket: URLSessionWebSocketTask?
    private var pendingMessages: [PendingNativeChatMessage] = []
    private var reconnectTask: Task<Void, Never>?
    private var launcherProcess: Process?
    private var attemptedLauncherRecovery = false

    init(port: Int) { self.port = port }

    func startGatewayAndConnect() {
        Task {
            var request = URLRequest(url: authenticatedConsoleURL(port: port, path: "/api/gateway/start"))
            request.httpMethod = "POST"
            _ = try? await URLSession.shared.data(for: request)
            connect()
        }
    }

    func connect() {
        guard socket == nil else { return }
        reconnectTask?.cancel()
        reconnectTask = nil
        status = "Connecting…"
        Task {
            do {
                var setup = URLRequest(url: authenticatedConsoleURL(port: port, path: "/api/jame/setup"))
                setup.httpMethod = "POST"
                let (data, _) = try await URLSession.shared.data(for: setup)
                let response = try JSONSerialization.jsonObject(with: data) as? [String: Any]
                guard let token = response?["token"] as? String,
                      let rawURL = response?["ws_url"] as? String,
                      var parts = URLComponents(string: rawURL) else { throw URLError(.badServerResponse) }
                var query = parts.queryItems ?? []
                query.append(URLQueryItem(name: "session_id", value: sessionID))
                parts.queryItems = query
                guard let wsURL = parts.url else { throw URLError(.badURL) }
                let task = URLSession.shared.webSocketTask(with: wsURL, protocols: ["token.\(token)"])
                socket = task
                task.resume()
                status = "Ready"
                flushPendingMessages()
                receive()
            } catch {
                socket = nil
                startBundledLauncherIfNeeded()
                status = "Waiting for JameClaw Desktop…"
                scheduleReconnect()
            }
        }
    }

    private func scheduleReconnect() {
        guard reconnectTask == nil else { return }
        reconnectTask = Task { [weak self] in
            try? await Task.sleep(for: .seconds(1))
            guard !Task.isCancelled else { return }
            self?.reconnectTask = nil
            self?.startGatewayAndConnect()
        }
    }

    // Jame.app is also useful when opened directly from Finder. In that case
    // its menu-bar launcher may not already be running, so recover by starting
    // the backend bundled alongside this app.
    private func startBundledLauncherIfNeeded() {
        guard !attemptedLauncherRecovery else { return }
        attemptedLauncherRecovery = true

        let resourcesDirectory = Bundle.main.bundleURL.deletingLastPathComponent()
        let contentsDirectory = resourcesDirectory.deletingLastPathComponent()
        let launcherURL = contentsDirectory
            .appendingPathComponent("MacOS", isDirectory: true)
            .appendingPathComponent("jameclaw-launcher")
        guard FileManager.default.isExecutableFile(atPath: launcherURL.path) else { return }

        let process = Process()
        process.executableURL = launcherURL
        process.arguments = ["-no-browser"]
        do {
            try process.run()
            launcherProcess = process
        } catch {
            // The launcher may already be running and holding the local port.
            // Retrying the connection below is still the correct recovery path.
        }
    }

    func send() {
        let content = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !content.isEmpty else { return }
        draft = ""
        let id = "native-\(UUID().uuidString)"
        messages.append(NativeChatMessage(id: id, role: "user", content: content))
        isThinking = true
        guard socket != nil else {
            pendingMessages.append(PendingNativeChatMessage(id: id, content: content))
            connect()
            return
        }
        send(id: id, content: content)
    }

    private func flushPendingMessages() {
        let queued = pendingMessages
        pendingMessages.removeAll()
        for message in queued {
            send(id: message.id, content: message.content)
        }
    }

    private func send(id: String, content: String) {
        let envelope: [String: Any] = ["type": "message.send", "id": id, "payload": ["content": content]]
        guard let data = try? JSONSerialization.data(withJSONObject: envelope),
              let text = String(data: data, encoding: .utf8) else {
            fail(messageID: id, message: "Could not prepare that message.")
            return
        }
        Task {
            do {
                try await socket?.send(.string(text))
            } catch {
                fail(messageID: id, message: "Message failed to send. Reconnecting…")
                socket = nil
                connect()
            }
        }
    }

    func sendMedia(data: Data, filename: String, contentType: String, kind: String) {
        guard socket != nil else {
            status = "Connecting to Jame. Try the upload again in a moment."
            connect()
            return
        }
        let id = "\(kind)-\(UUID().uuidString)"
        messages.append(NativeChatMessage(id: id, role: "user", content: "📎 \(filename)"))
        isThinking = true
        let envelope: [String: Any] = [
            "type": "media.send",
            "id": id,
            "payload": [
                "content": "",
                "data": data.base64EncodedString(),
                "filename": filename,
                "content_type": contentType,
                "kind": kind,
            ],
        ]
        guard let encoded = try? JSONSerialization.data(withJSONObject: envelope),
              let text = String(data: encoded, encoding: .utf8) else {
            fail(messageID: id, message: "Could not prepare that upload.")
            return
        }
        Task {
            do {
                try await socket?.send(.string(text))
            } catch {
                fail(messageID: id, message: "Upload failed. Reconnecting…")
                socket = nil
                connect()
            }
        }
    }

    private func fail(messageID: String, message: String) {
        messages.append(NativeChatMessage(id: "error-\(UUID().uuidString)", role: "error", content: message))
        status = message
        isThinking = false
    }

    private func receive() {
        Task {
            do {
                guard let message = try await socket?.receive() else { return }
                if case let .string(text) = message { handle(text) }
                receive()
            } catch {
                socket = nil
                status = "Reconnecting…"
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
        case "typing.stop": isThinking = false
        case "message.create":
            let content = payload["content"] as? String ?? ""
            let id = payload["message_id"] as? String ?? UUID().uuidString
            messages.append(NativeChatMessage(id: id, role: "assistant", content: content))
            isThinking = false
        case "message.update":
            let id = payload["message_id"] as? String ?? ""
            if let index = messages.firstIndex(where: { $0.id == id }) { messages[index].content = payload["content"] as? String ?? "" }
        case "error":
            messages.append(NativeChatMessage(id: UUID().uuidString, role: "error", content: (payload["message"] as? String) ?? "Request failed."))
            isThinking = false
        default: break
        }
    }
}

struct ChatView: View {
    @StateObject private var chat: NativeChatStore
    @AppStorage("launcher.design.theme") private var savedTheme = LauncherTheme.terminal.rawValue
    @AppStorage("launcher.design.fontScale") private var fontScale = 1.0
    @AppStorage("launcher.design.backgroundPath") private var backgroundPath = ""
    @State private var showingImagePicker = false
    @State private var isRecording = false
    @State private var recorder: AVAudioRecorder?
    @State private var recordingURL: URL?

    init(port: Int) { _chat = StateObject(wrappedValue: NativeChatStore(port: port)) }

    private var theme: LauncherTheme { LauncherTheme(rawValue: savedTheme) ?? .terminal }
    private var backgroundImage: NSImage? {
        guard !backgroundPath.isEmpty else { return nil }
        return NSImage(contentsOf: URL(fileURLWithPath: backgroundPath))
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 8) {
                Circle().fill(chat.status == "Ready" ? Color.green : Color.orange).frame(width: 7, height: 7)
                Text("JAME // CHAT").font(.system(size: 13 * fontScale, weight: .bold, design: .monospaced))
                Spacer()
                Text(chat.status.uppercased()).font(.system(size: 10 * fontScale, weight: .medium, design: .monospaced)).foregroundStyle(.secondary)
            }
            .foregroundStyle(theme.accent)
            .padding(.horizontal, 18).padding(.vertical, 13)
            .background(theme.panel)
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 12) {
                        if chat.messages.isEmpty {
                            VStack(alignment: .leading, spacing: 8) {
                                Text("jame@local:~$ ready for your prompt")
                                Text("Your conversation uses the same local Jame gateway as the web console.")
                                    .foregroundStyle(.secondary)
                            }
                            .font(.system(size: 14 * fontScale, design: .monospaced))
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.top, 8)
                        }
                        ForEach(chat.messages) { message in
                            VStack(alignment: .leading, spacing: 6) {
                                Text(message.role == "user" ? "you >" : message.role == "error" ? "error >" : "jame >")
                                    .font(.system(size: 10 * fontScale, weight: .bold, design: .monospaced))
                                    .foregroundStyle(message.role == "user" ? theme.accent : message.role == "error" ? .red : Color.green)
                                Text(message.content).textSelection(.enabled)
                            }
                                .font(.system(size: 14 * fontScale, design: .monospaced))
                                .foregroundStyle(theme.text)
                                .padding(12).frame(maxWidth: message.role == "user" ? 520 : .infinity, alignment: .leading)
                                .background(message.role == "user" ? theme.accent.opacity(theme == .light ? 0.14 : 0.22) : message.role == "error" ? Color.red.opacity(0.18) : Color.white.opacity(theme == .light ? 0.82 : 0.06))
                                .clipShape(RoundedRectangle(cornerRadius: 7)).frame(maxWidth: .infinity, alignment: message.role == "user" ? .trailing : .leading)
                                .id(message.id)
                        }
                        if chat.isThinking { Text("jame > thinking…").font(.system(size: 12 * fontScale, design: .monospaced)).foregroundStyle(Color.green) }
                    }.padding(18)
                }
                .background(chatBackground)
                .onChange(of: chat.messages.count) { _, _ in if let last = chat.messages.last { proxy.scrollTo(last.id, anchor: .bottom) } }
            }
            Divider().overlay(Color.white.opacity(0.12))
            HStack(alignment: .bottom) {
                Button {
                    showingImagePicker = true
                } label: {
                    Image(systemName: "photo")
                }
                .help("Attach an image")
                .disabled(chat.isThinking)

                Button {
                    toggleRecording()
                } label: {
                    Image(systemName: isRecording ? "stop.circle.fill" : "mic.fill")
                        .foregroundStyle(isRecording ? Color.red : theme.accent)
                }
                .help(isRecording ? "Stop and send recording" : "Record a voice message")
                .disabled(chat.isThinking && !isRecording)

                TextField("type a message…", text: $chat.draft, axis: .vertical)
                    .font(.system(size: 14 * fontScale, design: .monospaced)).lineLimit(1...5)
                    .textFieldStyle(.plain)
                Button("Send") { chat.send() }
                    .buttonStyle(.borderedProminent).tint(theme.accent)
                    .keyboardShortcut(.defaultAction)
            }
            .padding(14)
            .background(theme.panel)
        }
        .background(chatBackground)
        .preferredColorScheme(theme.colorScheme)
        .task { chat.startGatewayAndConnect() }
        .fileImporter(
            isPresented: $showingImagePicker,
            allowedContentTypes: [.image],
            allowsMultipleSelection: false
        ) { result in
            guard case let .success(urls) = result, let url = urls.first else { return }
            uploadImage(url)
        }
    }

    @ViewBuilder
    private var chatBackground: some View {
        ZStack {
            theme.background
            if let image = backgroundImage {
                Image(nsImage: image)
                    .resizable()
                    .scaledToFill()
                    .opacity(theme == .light ? 0.20 : 0.16)
                    .clipped()
            }
        }
    }

    private func uploadImage(_ url: URL) {
        let didAccess = url.startAccessingSecurityScopedResource()
        defer { if didAccess { url.stopAccessingSecurityScopedResource() } }
        do {
            let data = try Data(contentsOf: url)
            let contentType = UTType(filenameExtension: url.pathExtension)?.preferredMIMEType ?? "image/jpeg"
            chat.sendMedia(data: data, filename: url.lastPathComponent, contentType: contentType, kind: "image")
        } catch {
            chat.status = "Could not read that image."
        }
    }

    private func toggleRecording() {
        if isRecording {
            recorder?.stop()
            isRecording = false
            defer { recorder = nil }
            guard let url = recordingURL else { return }
            do {
                let data = try Data(contentsOf: url)
                chat.sendMedia(data: data, filename: "voice-\(Int(Date().timeIntervalSince1970)).m4a", contentType: "audio/mp4", kind: "audio")
            } catch {
                chat.status = "Could not read the recording."
            }
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
}
