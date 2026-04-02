import Foundation
import HandoffKit
import Observation
import OSLog

private let logger = Logger(subsystem: "sh.hitl.handoff", category: "LoopsManager")

@Observable
@MainActor
class LoopsManager {
    var loops: [APILoop] = []
    var isLoading = false
    var errorMessage: String?

    private let api = APIClient.shared

    // MARK: - Fetch Loops

    func fetchLoops() async {
        logger.info("fetchLoops: starting")
        isLoading = true
        errorMessage = nil
        do {
            loops = try await api.listLoops()
            logger.info("fetchLoops: got \(self.loops.count) loops: \(self.loops.map(\.name).joined(separator: ", "))")
            updateWidgetLoops()
        } catch {
            logger.error("fetchLoops: failed: \(error.localizedDescription)")
            errorMessage = error.localizedDescription
        }
        isLoading = false
    }

    // MARK: - Widget Loops

    private func updateWidgetLoops() {
        let loopInfos = loops.map { WidgetLoopInfo(id: $0.id, name: $0.name) }
        WidgetDataStore.writeLoops(loopInfos)
    }

    // MARK: - Join Loop

    func joinLoop(inviteCode: String) async throws -> APILoop {
        let loop = try await api.joinLoop(inviteCode: inviteCode)
        await fetchLoops()  // Refresh list
        return loop
    }

    // MARK: - Get Loop Detail

    func getLoopDetail(id: String) async throws -> LoopDetailData {
        try await api.getLoop(id: id)
    }

    // MARK: - Lookup by invite code (get details before joining)

    /// Looks up loop details without joining. The backend doesn't have a dedicated
    /// endpoint for this, so for now we just return basic info from the join response.
    /// In practice, the code view navigates to details and joins from there.
    func lookupLoop(inviteCode: String) async throws -> APILoop {
        // For now, we join directly. A future endpoint could support lookup without joining.
        try await api.joinLoop(inviteCode: inviteCode)
    }
}
