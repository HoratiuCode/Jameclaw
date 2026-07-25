import AppKit
import AVFoundation
import Foundation
import SwiftUI
import UniformTypeIdentifiers

extension Notification.Name {
    static let jameclawNewChat = Notification.Name("jameclaw.new-chat")
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
    var request = URLRequest(url: authenticatedConsoleURL(port: port, path: path, queryItems: queryItems))
    request.httpMethod = method
    let tokenURL = FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent(".jameclaw/launcher_access_token")
    if let token = try? String(contentsOf: tokenURL, encoding: .utf8).trimmingCharacters(in: .whitespacesAndNewlines), !token.isEmpty {
        request.setValue("jameclaw_launcher_session=\(token)", forHTTPHeaderField: "Cookie")
    }
    return request
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

    func applicationDidBecomeActive(_ notification: Notification) {
        revealMainWindow()
    }

    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        revealMainWindow()
        return true
    }

    func applicationShouldTerminate(_ sender: NSApplication) -> NSApplication.TerminateReply {
        var request = URLRequest(url: authenticatedConsoleURL(port: configuredLauncherPort(), path: "/api/system/quit"))
        request.httpMethod = "POST"
        let completed = DispatchSemaphore(value: 0)
        URLSession.shared.dataTask(with: request) { _, _, _ in completed.signal() }.resume()
        _ = completed.wait(timeout: .now() + 0.6)
        return .terminateNow
    }

    private func revealMainWindow() {
        DispatchQueue.main.async {
            guard let window = NSApp.windows.first else { return }
            window.center()
            window.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
        }
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
    case forest
    case lavender

    var id: String { rawValue }
    var label: String {
        switch self {
        case .terminal: return "Terminal"
        case .midnight: return "Midnight"
        case .light: return "Light"
        case .forest: return "Forest"
        case .lavender: return "Lavender"
        }
    }
    var colorScheme: ColorScheme { self == .light ? .light : .dark }
    var accent: Color {
        switch self {
        case .terminal: return Color(red: 0.95, green: 0.42, blue: 0.36)
        case .midnight: return Color(red: 0.36, green: 0.55, blue: 0.98)
        case .light: return Color(red: 0.72, green: 0.18, blue: 0.15)
        case .forest: return Color(red: 0.31, green: 0.78, blue: 0.53)
        case .lavender: return Color(red: 0.67, green: 0.52, blue: 0.98)
        }
    }
    var background: Color {
        switch self {
        case .terminal: return Color(red: 0.045, green: 0.05, blue: 0.055)
        case .midnight: return Color(red: 0.035, green: 0.06, blue: 0.12)
        case .light: return Color(red: 0.96, green: 0.97, blue: 0.98)
        case .forest: return Color(red: 0.025, green: 0.08, blue: 0.055)
        case .lavender: return Color(red: 0.09, green: 0.07, blue: 0.15)
        }
    }
    var panel: Color {
        switch self {
        case .terminal: return Color(red: 0.075, green: 0.08, blue: 0.09)
        case .midnight: return Color(red: 0.065, green: 0.10, blue: 0.19)
        case .light: return .white
        case .forest: return Color(red: 0.045, green: 0.13, blue: 0.09)
        case .lavender: return Color(red: 0.14, green: 0.11, blue: 0.23)
        }
    }
    var text: Color { self == .light ? Color(red: 0.13, green: 0.15, blue: 0.18) : Color(red: 0.9, green: 0.92, blue: 0.88) }
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
    @State private var selectedSection: DesktopSection? = .chat

    var body: some Scene {
        WindowGroup { JameRootView(selectedSection: $selectedSection) }
            // A content-sized window disables the standard macOS full-screen
            // control. Keep the desktop window resizable so it can enter
            // full screen from the title bar, the toolbar, or ⌃⌘F.
            .windowResizability(.automatic)
            .commands {
                CommandGroup(after: .newItem) {
                    Button("New Chat") {
                        selectedSection = .chat
                        NotificationCenter.default.post(name: .jameclawNewChat, object: nil)
                    }
                    .keyboardShortcut("n", modifiers: [.command])
                }

                CommandMenu("Automations") {
                    Button("Show Automations") {
                        selectedSection = .automations
                    }
                    .keyboardShortcut("a", modifiers: [.command, .option])
                }

                CommandMenu("View") {
                    ForEach(DesktopSection.allCases) { section in
                        Button(section.title) {
                            selectedSection = section
                        }
                        .keyboardShortcut(section.menuShortcut, modifiers: [.command, .option])
                    }
                }
            }
    }
}

struct JameRootView: View {
    @StateObject private var settings = LauncherSettingsStore()
    // The native launcher is a chat app first. Keep it selected on launch,
    // while retaining a persistent, desktop-native list of supporting views.
    @Binding var selectedSection: DesktopSection?

    var body: some View {
        NavigationSplitView {
            List(DesktopSection.allCases, selection: $selectedSection) { section in
                Label(section.title, systemImage: section.symbol)
                    .tag(section)
            }
            .navigationTitle("JameClaw")
            .listStyle(.sidebar)
        } detail: {
            switch selectedSection ?? .chat {
            case .chat:
                ChatView(port: Int(settings.port) ?? 18800)
            case .sessions:
                SessionsView(port: Int(settings.port) ?? 18800)
            case .automations:
                AutomationsView(port: Int(settings.port) ?? 18800)
            case .connectors:
                ConnectorsView(port: Int(settings.port) ?? 18800)
            case .artifacts:
                ArtifactsView()
            case .skills:
                SkillsView(port: Int(settings.port) ?? 18800)
            case .settings:
                QuickSettingsView(settings: settings)
            }
        }
        .navigationSplitViewStyle(.balanced)
        // Do not impose an application-level minimum or fixed content size.
        // This lets people size the Jame window however they prefer, including
        // narrow and compact layouts managed by macOS.
    }
}

enum DesktopSection: String, CaseIterable, Identifiable {
    case chat
    case artifacts
    case skills
    case sessions
    case automations
    case connectors
    case settings

    var id: Self { self }
    var title: String { rawValue.capitalized }
    var symbol: String {
        switch self {
        case .chat: return "message.fill"
        case .sessions: return "clock.arrow.circlepath"
        case .automations: return "calendar.badge.clock"
        case .connectors: return "point.3.connected.trianglepath.dotted"
        case .artifacts: return "shippingbox.fill"
        case .skills: return "wand.and.stars"
        case .settings: return "gearshape"
        }
    }

    var menuShortcut: KeyEquivalent {
        switch self {
        case .chat: return "1"
        case .artifacts: return "2"
        case .skills: return "3"
        case .sessions: return "4"
        case .automations: return "5"
        case .connectors: return "6"
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
                    Text("New automation").font(.title2.weight(.bold))
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
            Text("Set up \(blueprint.title)").font(.title2.weight(.bold))
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
                    .background(status == "Error" ? Color.red.opacity(0.15) : Color.secondary.opacity(0.12), in: Capsule())
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
                        .background(Color.secondary.opacity(0.08), in: RoundedRectangle(cornerRadius: 6))
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

    enum CodingKeys: String, CodingKey {
        case id, title, preview, updated, channel
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
        guard selectedSource != "All conversations" else { return sessions }
        return sessions.filter { sessionSourceName($0) == selectedSource }
    }

    func select(_ id: String?) async {
        selectedSessionID = id
        selectedSession = nil
        guard let id else { return }
        do {
            let path = "/api/sessions/\(id.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? id)"
            let (data, response) = try await URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: path))
            guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                throw URLError(.badServerResponse)
            }
            selectedSession = try JSONDecoder().decode(NativeSessionDetail.self, from: data)
            self.error = ""
        } catch {
            self.error = "Could not open this conversation."
        }
    }
}

private struct SessionsView: View {
    @StateObject private var store: NativeSessionStore

    init(port: Int) { _store = StateObject(wrappedValue: NativeSessionStore(port: port)) }

    var body: some View {
        HSplitView {
            VStack(spacing: 0) {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("Sessions").font(.title2.weight(.semibold))
                        Text("All conversations from Jame, Telegram, and connected channels")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    Picker("Conversation source", selection: $store.selectedSource) {
                        ForEach(store.sources, id: \.self) { source in
                            Text(source).tag(source)
                        }
                    }
                    .labelsHidden()
                    .pickerStyle(.menu)
                    Button { Task { await store.load() } } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                    .help("Refresh session history")
                }
                .padding()
                List(store.visibleSessions, selection: $store.selectedSessionID) { session in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(session.title.isEmpty ? session.preview : session.title)
                            .lineLimit(1)
                            .font(.headline)
                        Text("\(sessionSourceName(session)) · \(session.messageCount) messages · \(session.updated)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    .padding(.vertical, 3)
                    .tag(session.id)
                }
                .overlay {
                    if store.isLoading { ProgressView() }
                    else if store.visibleSessions.isEmpty {
                        ContentUnavailableView(
                            "No conversations",
                            systemImage: "clock",
                            description: Text(store.error.isEmpty ? "No saved conversations match this source." : store.error)
                        )
                    }
                }
            }
            .frame(minWidth: 285, idealWidth: 350)

            Group {
                if let session = store.selectedSession {
                    ScrollView {
                        LazyVStack(alignment: .leading, spacing: 14) {
                            ForEach(session.messages) { message in
                                VStack(alignment: .leading, spacing: 5) {
                                    Text(message.role.capitalized)
                                        .font(.caption.weight(.bold))
                                        .foregroundStyle(message.role == "user" ? .blue : .green)
                                    Text(message.content.isEmpty ? "(no text content)" : message.content)
                                        .textSelection(.enabled)
                                }
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(12)
                                .background(.quaternary, in: RoundedRectangle(cornerRadius: 10))
                            }
                        }
                        .padding()
                    }
                } else {
                    ContentUnavailableView("Select a session", systemImage: "bubble.left.and.bubble.right", description: Text("Choose a conversation to see its complete history."))
                }
            }
            .frame(minWidth: 420)
        }
        .task { await store.load() }
        .onChange(of: store.selectedSessionID) { _, id in Task { await store.select(id) } }
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
}

private struct ConnectorsView: View {
    @StateObject private var store: ConnectorsStore

    init(port: Int) { _store = StateObject(wrappedValue: ConnectorsStore(port: port)) }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Connectors").font(.title2.weight(.semibold))
                        Text("MCP servers and CLI providers available to this agent.")
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
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

                if store.isLoading { ProgressView("Loading connectors…") }
                if !store.error.isEmpty { Text(store.error).foregroundStyle(.red) }
            }
            .padding(24)
        }
        .task { await store.load() }
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
                .background(.quaternary, in: RoundedRectangle(cornerRadius: 12))
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
    private let port: Int

    init(port: Int) { self.port = port }

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                VStack(alignment: .leading, spacing: 3) {
                    Text("Skills").font(.title3.weight(.bold))
                    Text(browser.directory.path).font(.caption.monospaced()).foregroundStyle(.secondary).lineLimit(1)
                }
                Spacer()
                Button { refreshAgentSkills() } label: { Image(systemName: "arrow.clockwise") }
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
        .task { refreshAgentSkills() }
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

    private func refreshAgentSkills() {
        Task {
            do {
                let (data, response) = try await URLSession.shared.data(from: authenticatedConsoleURL(port: port, path: "/api/skills"))
                guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else { throw URLError(.badServerResponse) }
                let skillList = try JSONDecoder().decode(NativeSkillsResponse.self, from: data).skills
                let directories = skillList.map { URL(fileURLWithPath: $0.path).deletingLastPathComponent() }
                browser.entries = directories.map { WorkspaceEntry(url: $0, isDirectory: true) }
                browser.status = "\(skillList.count) skills available to the agent (workspace, global, and bundled)."
            } catch {
                browser.refresh()
                browser.status = "Showing workspace skills only. Could not reach the agent skill registry."
            }
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

    enum CodingKeys: String, CodingKey {
        case models
        case defaultModel = "default_model"
    }
}

private struct NativeProviderInfo: Codable {
    let name: String
    let configuredModels: [String]?

    enum CodingKeys: String, CodingKey {
        case name
        case configuredModels = "configured_models"
    }
}

private struct NativeProviderCatalogResponse: Codable {
    let providers: [NativeProviderInfo]
}

@MainActor
private final class NativeProviderStore: ObservableObject {
    @Published var models: [NativeModelInfo] = []
    @Published var defaultModel = ""
    @Published var selectedModel = ""
    @Published var providerNames: [String: String] = [:]
    @Published var status = ""
    @Published var isLoading = false

    func load(port: Int) async {
        isLoading = true
        defer { isLoading = false }

        do {
            async let modelsRequest: NativeModelsResponse = fetch(path: "/api/models", port: port)
            async let catalogRequest: NativeProviderCatalogResponse = fetch(path: "/api/models/catalog", port: port)
            let (modelsResponse, catalogResponse) = try await (modelsRequest, catalogRequest)
            models = modelsResponse.models.filter(\.configured)
            defaultModel = modelsResponse.defaultModel
            selectedModel = modelsResponse.defaultModel
            providerNames = Dictionary(uniqueKeysWithValues: catalogResponse.providers.flatMap { provider in
                (provider.configuredModels ?? []).map { ($0, provider.name) }
            })
            status = models.isEmpty ? "No configured AI providers yet." : ""
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
            defaultModel = modelName
            status = "Chat model updated. New chats will use it."
        } catch {
            status = "Could not change the chat model."
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

struct QuickSettingsView: View {
    @ObservedObject var settings: LauncherSettingsStore
    @StateObject private var providers = NativeProviderStore()
    @AppStorage("launcher.design.theme") private var savedTheme = LauncherTheme.terminal.rawValue
    @AppStorage("launcher.design.accent") private var savedAccent = LauncherAccent.theme.rawValue
    @AppStorage("launcher.design.density") private var savedDensity = ChatDensity.comfortable.rawValue
    @AppStorage("launcher.design.surface") private var savedSurface = MessageSurface.cards.rawValue
    @AppStorage("launcher.design.fontScale") private var fontScale = 1.0
    @AppStorage("launcher.design.backgroundPath") private var backgroundPath = ""
    @State private var showingBackgroundPicker = false
    @State private var allowOpenMacApps = false
    @State private var allowMusicPlaylists = false
    @State private var musicPlaylistStatus = ""

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
            Section("AI Provider") {
                if providers.models.isEmpty {
                    Text(providers.isLoading ? "Loading configured providers…" : "No configured AI providers.")
                        .foregroundStyle(.secondary)
                } else {
                    Picker("Chat model", selection: $providers.selectedModel) {
                        ForEach(providers.models) { model in
                            Text("\(providers.providerName(for: model)) · \(model.modelName)")
                                .tag(model.modelName)
                        }
                    }
                    Text("Current provider: \(providers.models.first(where: { $0.modelName == providers.defaultModel }).map { providers.providerName(for: $0) } ?? "Not selected")")
                        .font(.caption)
                        .foregroundStyle(.secondary)
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
                HStack {
                    Button("Choose chat background") { showingBackgroundPicker = true }
                    if !backgroundPath.isEmpty {
                        Button("Remove background", role: .destructive) { backgroundPath = "" }
                    }
                }
                Text("The selected image is stored locally and used behind the Chat view.")
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
            let port = Int(settings.port) ?? 18800
            await providers.load(port: port)
            await loadMusicPlaylistPermission(port: port)
        }
        .onChange(of: providers.selectedModel) { _, modelName in
            guard !modelName.isEmpty else { return }
            Task { await providers.setDefaultModel(modelName, port: Int(settings.port) ?? 18800) }
        }
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

private struct PendingNativeChatMessage {
    let id: String
    let content: String
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
    @Published var draft = ""
    @Published var status = "Connecting…"
    @Published var isThinking = false
    @Published var lastError: NativeAppError?
    @Published var workspaceName = "Choose workspace"

    private let port: Int
    private let sessionID: String
    private var socket: URLSessionWebSocketTask?
    private var pendingMessages: [PendingNativeChatMessage] = []
    private var reconnectTask: Task<Void, Never>?
    private var launcherProcess: Process?
    private var attemptedLauncherRecovery = false
    private var reconnectAttempt = 0

    init(port: Int) {
        self.port = port
        workspaceName = jameWorkspaceURL().lastPathComponent
        let key = "jameclaw.native-chat.session-id"
        if let storedID = UserDefaults.standard.string(forKey: key), !storedID.isEmpty {
            sessionID = storedID
        } else {
            let newID = UUID().uuidString
            UserDefaults.standard.set(newID, forKey: key)
            sessionID = newID
        }
    }

    func startGatewayAndConnect() {
        Task {
            do {
                let request = authenticatedConsoleRequest(port: port, path: "/api/gateway/start", method: "POST")
                let (_, response) = try await URLSession.shared.data(for: request)
                guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                    throw URLError(.badServerResponse)
                }
                try await waitForGatewayReadiness()
                connect()
            } catch {
                startBundledLauncherIfNeeded()
                reportError(title: "JameClaw is not ready", detail: connectionDetail(for: error))
                scheduleReconnect()
            }
        }
    }

    func connect() {
        guard socket == nil else { return }
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
                status = "Ready"
                lastError = nil
                reconnectAttempt = 0
                flushPendingMessages()
                receive()
            } catch {
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
        reconnectTask?.cancel()
        reconnectTask = nil
        socket?.cancel(with: .goingAway, reason: nil)
        socket = nil
        lastError = nil
        reconnectAttempt = 0
        startGatewayAndConnect()
    }

    func dismissError() { lastError = nil }

    func startNewChat() {
        messages.removeAll()
        pendingMessages.removeAll()
        draft = ""
        isThinking = false
        lastError = nil
        status = socket == nil ? "Connecting…" : "Ready"
    }

    func setWorkspace(_ workspaceURL: URL) {
        let workspacePath = workspaceURL.standardizedFileURL.path
        Task {
            do {
                status = "Updating workspace…"
                var request = URLRequest(url: authenticatedConsoleURL(port: port, path: "/api/config"))
                request.httpMethod = "PATCH"
                request.setValue("application/json", forHTTPHeaderField: "Content-Type")
                request.httpBody = try JSONSerialization.data(withJSONObject: [
                    "agents": ["defaults": ["workspace": workspacePath]],
                ])
                let (_, response) = try await URLSession.shared.data(for: request)
                guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                    throw URLError(.badServerResponse)
                }

                var restart = URLRequest(url: authenticatedConsoleURL(port: port, path: "/api/gateway/restart"))
                restart.httpMethod = "POST"
                let (_, restartResponse) = try await URLSession.shared.data(for: restart)
                guard let http = restartResponse as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                    throw URLError(.badServerResponse)
                }

                workspaceName = workspaceURL.lastPathComponent
                status = "Workspace updated. Reconnecting…"
                socket?.cancel(with: .goingAway, reason: nil)
                socket = nil
                lastError = nil
                scheduleReconnect()
            } catch {
                reportError(title: "Could not change workspace", detail: connectionDetail(for: error))
            }
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
        let outboundContent = nativeAppCommandInstruction(for: content)
        guard socket != nil else {
            pendingMessages.append(PendingNativeChatMessage(id: id, content: outboundContent))
            status = "Connecting…"
            connect()
            return
        }
        send(id: id, content: outboundContent)
    }

    func sendSkillImported(_ skillName: String) {
        draft = "I uploaded the \(skillName) skill to this workspace. Please read and use it for this task."
        send()
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
                guard let socket else { throw URLError(.notConnectedToInternet) }
                try await socket.send(.string(text))
            } catch {
                fail(messageID: id, message: "Message failed to send. Reconnecting…")
                socket = nil
                connect()
            }
        }
    }

    func sendMedia(data: Data, filename: String, contentType: String, kind: String, content: String = "") {
        guard socket != nil else {
            status = "Connecting to Jame. Try the upload again in a moment."
            connect()
            return
        }
        let id = "\(kind)-\(UUID().uuidString)"
        let displayContent = content.isEmpty ? "📎 \(filename)" : "\(content)\n📎 \(filename)"
        messages.append(NativeChatMessage(id: id, role: "user", content: displayContent))
        isThinking = true
        let envelope: [String: Any] = [
            "type": "media.send",
            "id": id,
            "payload": [
                "content": content,
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
        status = title
        isThinking = false
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

    private func receive() {
        Task {
            do {
                guard let message = try await socket?.receive() else { return }
                if case let .string(text) = message { handle(text) }
                receive()
            } catch {
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
        case "typing.stop": isThinking = false
        case "message.create":
            let content = responseContent(from: payload)
            let id = (payload["message_id"] as? String) ?? (event["id"] as? String) ?? UUID().uuidString
            upsertAssistantMessage(id: id, content: content)
            isThinking = false
        case "message.update":
            let id = (payload["message_id"] as? String) ?? (event["id"] as? String) ?? UUID().uuidString
            // A gateway can begin streaming before its placeholder reaches the
            // desktop. Treat that update as the first visible assistant reply
            // instead of dropping it.
            upsertAssistantMessage(id: id, content: responseContent(from: payload))
        case "error":
            let message = responseContent(from: payload, fallback: "The provider could not complete this request.")
            messages.append(NativeChatMessage(id: UUID().uuidString, role: "error", content: message))
            reportError(title: "Provider error", detail: message)
        default: break
        }
    }

    private func responseContent(from payload: [String: Any], fallback: String = "") -> String {
        for key in ["content", "text", "message", "error"] {
            if let value = payload[key] as? String, !value.isEmpty { return value }
        }
        return fallback
    }

    private func upsertAssistantMessage(id: String, content: String) {
        if let index = messages.firstIndex(where: { $0.id == id }) {
            messages[index].content = content
        } else {
            messages.append(NativeChatMessage(id: id, role: "assistant", content: content))
        }
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
    enum Kind { case app, skill, file }
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
    @StateObject private var chat: NativeChatStore
    private let port: Int
    @AppStorage("launcher.design.theme") private var savedTheme = LauncherTheme.terminal.rawValue
    @AppStorage("launcher.design.accent") private var savedAccent = LauncherAccent.theme.rawValue
    @AppStorage("launcher.design.density") private var savedDensity = ChatDensity.comfortable.rawValue
    @AppStorage("launcher.design.surface") private var savedSurface = MessageSurface.cards.rawValue
    @AppStorage("launcher.design.fontScale") private var fontScale = 1.0
    @AppStorage("launcher.design.backgroundPath") private var backgroundPath = ""
    @State private var isRecording = false
    @State private var recorder: AVAudioRecorder?
    @State private var recordingURL: URL?
    @State private var suggestions: [ChatComposerSuggestion] = []
    @State private var appCommands: [ChatComposerSuggestion] = []
    @State private var pendingAttachment: PendingChatAttachment?

    init(port: Int) {
        self.port = port
        _chat = StateObject(wrappedValue: NativeChatStore(port: port))
    }

    private var theme: LauncherTheme { LauncherTheme(rawValue: savedTheme) ?? .terminal }
    private var accent: Color { (LauncherAccent(rawValue: savedAccent) ?? .theme).color ?? theme.accent }
    private var density: ChatDensity { ChatDensity(rawValue: savedDensity) ?? .comfortable }
    private var messageSurface: MessageSurface { MessageSurface(rawValue: savedSurface) ?? .cards }
    private var backgroundImage: NSImage? {
        guard !backgroundPath.isEmpty else { return nil }
        return NSImage(contentsOf: URL(fileURLWithPath: backgroundPath))
    }
    private var isConnectingToJame: Bool {
        chat.status != "Ready" && chat.messages.isEmpty && chat.lastError == nil
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 8) {
                Text("Jame.")
                    .font(.system(size: 15 * fontScale, weight: .bold, design: .rounded))
                    .foregroundStyle(accent)
                Spacer()
                Text(chat.status.uppercased()).font(.system(size: 10 * fontScale, weight: .medium, design: .monospaced)).foregroundStyle(.secondary)
            }
            .foregroundStyle(accent)
            .padding(.horizontal, 18).padding(.vertical, 13)
            .background(theme.panel)
            Button {
                chooseWorkspace()
            } label: {
                HStack(spacing: 8) {
                    Image(systemName: "folder.fill")
                        .foregroundStyle(accent)
                    VStack(alignment: .leading, spacing: 1) {
                        Text("Workspace")
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(.secondary)
                        Text(chat.workspaceName)
                            .font(.subheadline.weight(.medium))
                            .lineLimit(1)
                    }
                    Spacer()
                    Image(systemName: "chevron.right")
                        .font(.caption.weight(.bold))
                        .foregroundStyle(.secondary)
                }
                .padding(.horizontal, 18)
                .padding(.vertical, 9)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .help("Choose agent workspace")
            .background(theme.panel.opacity(0.9))
            if let error = chat.lastError {
                HStack(alignment: .top, spacing: 10) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundStyle(.orange)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(error.title).font(.subheadline.weight(.semibold))
                        Text(error.detail).font(.caption).foregroundStyle(.secondary)
                    }
                    Spacer(minLength: 8)
                    Button("Retry") { chat.retryConnection() }
                        .buttonStyle(.bordered)
                        .controlSize(.small)
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
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: density.messageSpacing) {
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
                                    .foregroundStyle(message.role == "user" ? accent : message.role == "error" ? .red : Color.green)
                                Text(message.content).textSelection(.enabled)
                            }
                                .font(.system(size: 14 * fontScale, design: .monospaced))
                                .foregroundStyle(theme.text)
                                .padding(density.messagePadding).frame(maxWidth: message.role == "user" ? 520 : .infinity, alignment: .leading)
                                .background(messageSurface == .cards ? (message.role == "user" ? accent.opacity(theme == .light ? 0.14 : 0.22) : message.role == "error" ? Color.red.opacity(0.18) : Color.white.opacity(theme == .light ? 0.82 : 0.06)) : .clear)
                                .clipShape(RoundedRectangle(cornerRadius: messageSurface == .cards ? 8 : 0)).frame(maxWidth: .infinity, alignment: message.role == "user" ? .trailing : .leading)
                                .id(message.id)
                        }
                        if chat.isThinking { Text("jame > thinking…").font(.system(size: 12 * fontScale, design: .monospaced)).foregroundStyle(Color.green) }
                    }.padding(density.contentPadding)
                }
                .background(chatBackground)
                .onChange(of: chat.messages.count) { _, _ in if let last = chat.messages.last { proxy.scrollTo(last.id, anchor: .bottom) } }
            }
            Divider().overlay(Color.white.opacity(0.12))
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
                .background(theme.panel.opacity(0.92))
            }
            ZStack(alignment: .bottomLeading) {
                HStack(alignment: .bottom) {
                    Button {
                        chooseWorkspace()
                    } label: {
                        Image(systemName: "folder")
                    }
                    .help("Choose agent workspace")
                    .disabled(chat.isThinking)

                    Button {
                        uploadItem()
                    } label: {
                        Image(systemName: "paperclip")
                    }
                    .help("Upload a file or workspace skill")
                    .disabled(chat.isThinking)

                    Button {
                        toggleRecording()
                    } label: {
                        Image(systemName: isRecording ? "stop.circle.fill" : "mic.fill")
                            .foregroundStyle(isRecording ? Color.red : accent)
                    }
                    .help(isRecording ? "Stop and send recording" : "Record a voice message")
                    .disabled(chat.isThinking && !isRecording)

                    TextField("type a message…", text: $chat.draft, axis: .vertical)
                        .font(.system(size: 14 * fontScale, design: .monospaced)).lineLimit(1...5)
                        .textFieldStyle(.plain)
                        .onChange(of: chat.draft) { _, value in updateSuggestions(for: value) }
                    Button("Send") { sendComposer() }
                        .buttonStyle(.borderedProminent).tint(accent)
                        .keyboardShortcut(.defaultAction)
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
                    .background(theme.panel.opacity(0.98))
                    .clipShape(RoundedRectangle(cornerRadius: 10))
                    .overlay(RoundedRectangle(cornerRadius: 10).stroke(Color.white.opacity(0.14)))
                    .shadow(color: .black.opacity(0.24), radius: 10, y: 4)
                    .offset(x: 56, y: -70)
                    .zIndex(1)
                }
            }
            .background(theme.panel)
        }
        .background(chatBackground)
        .preferredColorScheme(theme.colorScheme)
        .overlay {
            if isConnectingToJame {
                jameLoadingScreen
            }
        }
        .task {
            appCommands = desktopAppCommands()
            chat.startGatewayAndConnect()
        }
        .onReceive(NotificationCenter.default.publisher(for: .jameclawNewChat)) { _ in
            chat.startNewChat()
        }
    }

    private var jameLoadingScreen: some View {
        ZStack {
            chatBackground
            VStack(spacing: 16) {
                Text("Jame")
                    .font(.system(size: 34 * fontScale, weight: .bold, design: .rounded))
                    .foregroundStyle(accent)
                ProgressView()
                    .controlSize(.large)
                    .tint(accent)
                Text("Connecting to JameClaw on localhost…")
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
                Image(nsImage: image)
                    .resizable()
                    .scaledToFill()
                    .opacity(theme == .light ? 0.20 : 0.16)
                    .clipped()
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

    private func sendComposer() {
        let content = chat.draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !content.isEmpty || pendingAttachment != nil else { return }
        guard let attachment = pendingAttachment else {
            chat.send()
            return
        }

        chat.draft = ""
        pendingAttachment = nil
        chat.sendMedia(
            data: attachment.data,
            filename: attachment.filename,
            contentType: attachment.contentType,
            kind: attachment.kind,
            content: content
        )
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
        panel.title = "Choose JameClaw Workspace"
        panel.message = "Choose the folder where Jame should read and create workspace files."
        panel.prompt = "Use Workspace"
        panel.canChooseFiles = false
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false
        panel.canCreateDirectories = true

        guard panel.runModal() == .OK, let workspaceURL = panel.url else { return }
        chat.setWorkspace(workspaceURL)
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
