import Foundation

/// Owns the bundled launcher process for the lifetime of the visible macOS
/// app. Keeping this outside SwiftUI views prevents duplicate starts and
/// guarantees that a helper started by Desktop is stopped with Desktop.
@MainActor
final class NativeLaunchCoordinator: ObservableObject {
    enum State: Equatable {
        case stopped
        case starting
        case running
        case unavailable
    }

    static let shared = NativeLaunchCoordinator()

    @Published private(set) var state: State = .stopped
    private var process: Process?
    private var restartTask: Task<Void, Never>?
    private var stableLaunchTask: Task<Void, Never>?
    private var restartAttempt = 0
    private var isStopping = false

    func startIfNeeded() {
        guard process?.isRunning != true, state != .starting else { return }
        restartTask?.cancel()
        restartTask = nil
        isStopping = false

        let bundleURL = Bundle.main.bundleURL
        let contentsURL = bundleURL.deletingLastPathComponent()
        let candidates = [
            bundleURL.appendingPathComponent("Contents/MacOS/jameclaw-launcher"),
            contentsURL.appendingPathComponent("MacOS/jameclaw-launcher"),
        ]
        guard let launcherURL = candidates.first(where: {
            FileManager.default.isExecutableFile(atPath: $0.path)
        }) else {
            state = .unavailable
            return
        }

        state = .starting
        let launcher = Process()
        launcher.executableURL = launcherURL
        launcher.arguments = ["-no-browser"]
        launcher.terminationHandler = { [weak self, weak launcher] exitedProcess in
            DispatchQueue.main.async {
                guard let self, self.process === launcher else { return }
                self.process = nil
                guard !self.isStopping else {
                    self.state = .stopped
                    return
                }
                self.scheduleRestart(afterUnexpectedExit: exitedProcess.terminationStatus)
            }
        }
        do {
            try launcher.run()
            process = launcher
            state = .running
            // Do not reset the crash budget immediately: a helper that exits
            // right after launch would otherwise restart forever. A stable
            // half minute of runtime earns a fresh retry budget.
            stableLaunchTask?.cancel()
            stableLaunchTask = Task { [weak self] in
                try? await Task.sleep(for: .seconds(30))
                guard !Task.isCancelled else { return }
                self?.restartAttempt = 0
            }
        } catch {
            process = nil
            state = .unavailable
        }
    }

    func stop() {
        isStopping = true
        restartTask?.cancel()
        restartTask = nil
        stableLaunchTask?.cancel()
        stableLaunchTask = nil
        guard let process else { return }
        if process.isRunning { process.terminate() }
        self.process = nil
        state = .stopped
    }

    /// A launcher crash should not leave a perfectly healthy-looking desktop
    /// window connected to nothing. Retry a few times; NativeChatStore keeps
    /// the user-facing connection status and will surface a recovery action.
    private func scheduleRestart(afterUnexpectedExit status: Int32) {
        stableLaunchTask?.cancel()
        stableLaunchTask = nil
        guard restartAttempt < 3 else {
            state = .unavailable
            return
        }
        let delay = min(pow(2, Double(restartAttempt)), 8)
        restartAttempt += 1
        state = .starting
        restartTask?.cancel()
        restartTask = Task { [weak self] in
            try? await Task.sleep(for: .seconds(delay))
            guard !Task.isCancelled else { return }
            guard let self else { return }
            self.restartTask = nil
            // startIfNeeded intentionally coalesces concurrent starts while
            // state is .starting. This delayed recovery is now the sole
            // starter, so release that marker before invoking it.
            self.state = .stopped
            self.startIfNeeded()
        }
        NSLog("[JameLauncher] exited unexpectedly (status=%d); retrying in %.0fs", status, delay)
    }
}
