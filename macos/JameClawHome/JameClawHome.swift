import AppKit
import Foundation
import SwiftUI

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
    @State private var selectedTab = 0

    var body: some View {
        TabView(selection: $selectedTab) {
            HomeView(settings: settings) { selectedTab = 2 }
                .tabItem { Label("Home", systemImage: "sparkles") }
                .tag(0)
            ChatView(port: Int(settings.port) ?? 18800)
                .tabItem { Label("Chat", systemImage: "message.fill") }
                .tag(1)
            QuickSettingsView(settings: settings)
                .tabItem { Label("Settings", systemImage: "gearshape") }
                .tag(2)
        }
        .frame(width: 760, height: 620)
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
        }
        .formStyle(.grouped)
        .padding(.top, 6)
    }
}

struct NativeChatMessage: Identifiable {
    let id: String
    let role: String
    var content: String
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

    init(port: Int) { self.port = port }

    func connect() {
        guard socket == nil else { return }
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
                receive()
            } catch {
                status = "Jame is still starting. Try again shortly."
            }
        }
    }

    func send() {
        let content = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !content.isEmpty else { return }
        if socket == nil { connect(); return }
        draft = ""
        let id = "native-\(UUID().uuidString)"
        messages.append(NativeChatMessage(id: id, role: "user", content: content))
        isThinking = true
        let envelope: [String: Any] = ["type": "message.send", "id": id, "payload": ["content": content]]
        guard let data = try? JSONSerialization.data(withJSONObject: envelope), let text = String(data: data, encoding: .utf8) else { return }
        Task { try? await socket?.send(.string(text)) }
    }

    private func receive() {
        Task {
            do {
                guard let message = try await socket?.receive() else { return }
                if case let .string(text) = message { handle(text) }
                receive()
            } catch {
                socket = nil
                status = "Connection lost."
                isThinking = false
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

    init(port: Int) { _chat = StateObject(wrappedValue: NativeChatStore(port: port)) }

    var body: some View {
        VStack(spacing: 0) {
            HStack { Text("Jame chat").font(.headline); Spacer(); Text(chat.status).font(.caption).foregroundStyle(.secondary) }
                .padding(.horizontal, 18).padding(.vertical, 12)
            Divider()
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 12) {
                        ForEach(chat.messages) { message in
                            Text(message.content).textSelection(.enabled)
                                .font(.system(.body, design: .monospaced))
                                .padding(12).frame(maxWidth: message.role == "user" ? 520 : .infinity, alignment: .leading)
                                .background(message.role == "user" ? Color.red.opacity(0.14) : message.role == "error" ? Color.red.opacity(0.22) : Color.secondary.opacity(0.10))
                                .clipShape(RoundedRectangle(cornerRadius: 10)).frame(maxWidth: .infinity, alignment: message.role == "user" ? .trailing : .leading)
                                .id(message.id)
                        }
                        if chat.isThinking { Text("Jame is thinking…").font(.caption).foregroundStyle(.secondary) }
                    }.padding(18)
                }.onChange(of: chat.messages.count) { _, _ in if let last = chat.messages.last { proxy.scrollTo(last.id, anchor: .bottom) } }
            }
            Divider()
            HStack(alignment: .bottom) {
                TextField("Message Jame…", text: $chat.draft, axis: .vertical).lineLimit(1...5)
                Button("Send") { chat.send() }.keyboardShortcut(.defaultAction)
            }.padding(14)
        }
        .task { chat.connect() }
    }
}
