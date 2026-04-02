import Foundation
import HandoffKit
import Observation

@Observable
@MainActor
class RequestsManager {
    var pendingRequests: [APIRequest] = []
    var historyRequests: [APIRequest] = []
    var isLoadingPending = false
    var isLoadingHistory = false
    var errorMessage: String?

    /// Set by HandoffApp to update Live Activities when requests change.
    var liveActivityManager: LiveActivityManager?

    private let api = APIClient.shared

    // MARK: - Fetch Pending Requests

    func fetchPending() async {
        isLoadingPending = true
        errorMessage = nil
        do {
            let data = try await api.listRequests(status: "pending", sort: "priority_desc")
            pendingRequests = data.requests
            // Update the app-wide Live Activity with the latest data
            await liveActivityManager?.updateActivity(allPending: pendingRequests)
            updateWidgetData()
        } catch {
            errorMessage = error.localizedDescription
        }
        isLoadingPending = false
    }

    // MARK: - Fetch History (completed, timeout, cancelled)

    func fetchHistory() async {
        isLoadingHistory = true
        errorMessage = nil
        do {
            // Fetch non-pending requests
            let completed = try await api.listRequests(status: "completed", sort: "created_at_desc")
            let timedOut = try await api.listRequests(status: "timeout", sort: "created_at_desc")
            let cancelled = try await api.listRequests(status: "cancelled", sort: "created_at_desc")
            historyRequests = (completed.requests + timedOut.requests + cancelled.requests)
                .sorted { $0.createdAt > $1.createdAt }
        } catch {
            errorMessage = error.localizedDescription
        }
        isLoadingHistory = false
    }

    // MARK: - Respond to Request

    func respond(requestId: String, responseData: Any) async throws {
        _ = try await api.respondToRequest(id: requestId, responseData: responseData)
        pendingRequests.removeAll { $0.id == requestId }

        // Update the app-wide Live Activity
        await liveActivityManager?.updateActivity(allPending: pendingRequests)
        updateWidgetData()
    }

    // MARK: - Cancel Request

    func cancel(requestId: String) async throws {
        try await api.cancelRequest(id: requestId)
        pendingRequests.removeAll { $0.id == requestId }

        // Update the app-wide Live Activity
        await liveActivityManager?.updateActivity(allPending: pendingRequests)
        updateWidgetData()
    }

    // MARK: - Get single request

    func getRequest(id: String) async throws -> APIRequest {
        try await api.getRequest(id: id)
    }

    // MARK: - Widget Data

    private func updateWidgetData() {
        let sorted = pendingRequests
            .sorted { r1, r2 in
                if r1.priority != r2.priority { return r1.priority == "urgent" }
                return (r1.timeoutDate ?? .distantFuture) < (r2.timeoutDate ?? .distantFuture)
            }

        // Find the loop with the most pending requests
        var loopCounts: [String: (name: String, count: Int)] = [:]
        for req in pendingRequests {
            let name = req.loopNameDisplay
            let existing = loopCounts[req.loopId] ?? (name: name, count: 0)
            loopCounts[req.loopId] = (name: existing.name, count: existing.count + 1)
        }
        let topLoop = loopCounts.values.max(by: { $0.count < $1.count })

        let summaries = sorted.prefix(5).map { req in
            WidgetData.PendingRequestSummary(
                id: req.id, loopId: req.loopId, text: req.displayText(),
                loopName: req.loopNameDisplay,
                responseType: req.responseType, priority: req.priority,
                timeoutAt: req.timeoutDate
            )
        }

        let widgetData = WidgetData(
            totalPending: pendingRequests.count,
            topRequests: Array(summaries),
            topLoopName: topLoop?.name,
            topLoopPendingCount: topLoop?.count ?? 0,
            lastUpdated: Date()
        )

        // Merge loops from pending requests into widget loops list
        let requestLoops = loopCounts.map { WidgetLoopInfo(id: $0.key, name: $0.value.name) }
        let existingLoops = WidgetDataStore.readLoops()
        var mergedById: [String: WidgetLoopInfo] = [:]
        for loop in existingLoops { mergedById[loop.id] = loop }
        for loop in requestLoops { mergedById[loop.id] = loop }
        WidgetDataStore.writeLoops(Array(mergedById.values))

        WidgetDataStore.write(widgetData)
        WidgetDataStore.reloadWidgets()
    }
}
