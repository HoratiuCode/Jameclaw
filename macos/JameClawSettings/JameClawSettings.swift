import AppKit
import SwiftUI

private func authenticatedConsoleURL(port: Int) -> URL {
    var components = URLComponents()
    components.scheme = "http"
    components.host = "localhost"
    components.port = port
    let tokenURL = FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent(".jameclaw/launcher_access_token")
    if let token = try? String(contentsOf: tokenURL, encoding: .utf8).trimmingCharacters(in: .whitespacesAndNewlines), !token.isEmpty {
        components.queryItems = [URLQueryItem(name: "access_token", value: token)]
    }
    return components.url ?? URL(string: "http://localhost:\(port)")!
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
final class SettingsStore: ObservableObject {
    @Published var port = "18800"
    @Published var lanAccess = false
    @Published var allowedCIDRs = ""
    @Published var status = ""

    private let configURL = FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent(".jameclaw/launcher-config.json")

    init() { load() }

    func load() {
        guard let data = try? Data(contentsOf: configURL),
              let settings = try? JSONDecoder().decode(LauncherSettings.self, from: data) else { return }
        port = String(settings.port)
        lanAccess = settings.public
        allowedCIDRs = settings.allowedCIDRs.joined(separator: "\n")
        status = "Loaded current launcher settings."
    }

    func save() {
        guard let portNumber = Int(port), (1...65535).contains(portNumber) else {
            status = "Enter a port between 1 and 65535."
            return
        }
        let cidrs = allowedCIDRs.split(whereSeparator: \.isNewline).map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }
        let settings = LauncherSettings(port: portNumber, public: lanAccess, allowedCIDRs: cidrs)
        do {
            try FileManager.default.createDirectory(at: configURL.deletingLastPathComponent(), withIntermediateDirectories: true)
            let data = try JSONEncoder.pretty.encode(settings)
            try data.write(to: configURL, options: .atomic)
            status = "Saved. Restart JameClaw for network changes to take effect."
        } catch {
            status = "Could not save settings: \(error.localizedDescription)"
        }
    }

    func openConsole() {
        let targetPort = Int(port) ?? 18800
        NSWorkspace.shared.open(authenticatedConsoleURL(port: targetPort))
    }
}

private extension JSONEncoder {
    static var pretty: JSONEncoder {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        return encoder
    }
}

struct SettingsView: View {
    @StateObject private var store = SettingsStore()

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            HStack(spacing: 10) {
                Image(systemName: "gearshape.fill").foregroundStyle(.red)
                VStack(alignment: .leading, spacing: 2) {
                    Text("Jame Settings").font(.title3.weight(.semibold))
                    Text("Simple launcher preferences").foregroundStyle(.secondary)
                }
            }

            Form {
                TextField("Web Console port", text: $store.port)
                Toggle("Allow devices on my local network", isOn: $store.lanAccess)
                if store.lanAccess {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Allowed network CIDRs (one per line)").font(.caption).foregroundStyle(.secondary)
                        TextEditor(text: $store.allowedCIDRs).font(.system(.body, design: .monospaced)).frame(height: 78)
                    }
                }
            }
            .formStyle(.grouped)

            Text("Use the Web Console for models, channels, tools, and advanced settings.")
                .font(.footnote).foregroundStyle(.secondary)
            if !store.status.isEmpty { Text(store.status).font(.footnote).foregroundStyle(.secondary) }

            HStack {
                Button("Open Web Console") { store.openConsole() }
                Spacer()
                Button("Save settings") { store.save() }.keyboardShortcut(.defaultAction)
            }

            Text("Developed by Jame")
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity, alignment: .center)
        }
        .padding(22)
        // The settings window should follow the user's chosen size too.
    }
}

@main
struct JameClawSettingsApp: App {
    var body: some Scene {
        WindowGroup { SettingsView() }
            .windowResizability(.automatic)
    }
}
