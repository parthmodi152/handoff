//
//  WidgetData.swift
//  HandoffKit
//
//  Shared data model for Home Screen & Lock Screen widgets.
//  Written by the main app, read by the widget extension via App Groups UserDefaults.
//

import Foundation

// MARK: - Loop Info (for widget configuration)

/// Lightweight loop reference stored in shared UserDefaults for the widget to display
/// in its configuration picker.
public struct WidgetLoopInfo: Codable, Identifiable, Hashable {
    public let id: String
    public let name: String

    public init(id: String, name: String) {
        self.id = id
        self.name = name
    }
}

// MARK: - Widget Data

public struct WidgetData: Codable {
    public struct PendingRequestSummary: Codable, Identifiable {
        public let id: String
        public let loopId: String
        public let text: String
        public let loopName: String
        public let responseType: String
        public let priority: String
        public let timeoutAt: Date?

        public init(id: String, loopId: String, text: String, loopName: String,
                    responseType: String, priority: String, timeoutAt: Date?) {
            self.id = id
            self.loopId = loopId
            self.text = String(text.prefix(60))
            self.loopName = loopName
            self.responseType = responseType
            self.priority = priority
            self.timeoutAt = timeoutAt
        }
    }

    public let totalPending: Int
    public let topRequests: [PendingRequestSummary]
    public let topLoopName: String?
    public let topLoopPendingCount: Int
    public let lastUpdated: Date

    public init(totalPending: Int, topRequests: [PendingRequestSummary],
                topLoopName: String?, topLoopPendingCount: Int, lastUpdated: Date) {
        self.totalPending = totalPending
        self.topRequests = Array(topRequests.prefix(5))
        self.topLoopName = topLoopName
        self.topLoopPendingCount = topLoopPendingCount
        self.lastUpdated = lastUpdated
    }

    /// Filter this data to only include requests for a specific loop.
    public func filtered(byLoopId loopId: String) -> WidgetData {
        let filtered = topRequests.filter { $0.loopId == loopId }
        let loopName = filtered.first?.loopName
        let totalForLoop = filtered.count
        return WidgetData(
            totalPending: totalForLoop,
            topRequests: filtered,
            topLoopName: loopName,
            topLoopPendingCount: totalForLoop,
            lastUpdated: lastUpdated
        )
    }

    public static let empty = WidgetData(
        totalPending: 0, topRequests: [], topLoopName: nil,
        topLoopPendingCount: 0, lastUpdated: Date()
    )

    public static let preview = WidgetData(
        totalPending: 7,
        topRequests: [
            .init(id: "1", loopId: "loop-1", text: "Approve prod deployment", loopName: "DevOps",
                  responseType: "boolean", priority: "urgent",
                  timeoutAt: Date().addingTimeInterval(154)),
            .init(id: "2", loopId: "loop-1", text: "Review PR #482", loopName: "DevOps",
                  responseType: "single_select", priority: "normal",
                  timeoutAt: Date().addingTimeInterval(300)),
            .init(id: "3", loopId: "loop-2", text: "Confirm budget increase", loopName: "Finance",
                  responseType: "text", priority: "normal", timeoutAt: nil),
        ],
        topLoopName: "DevOps", topLoopPendingCount: 5, lastUpdated: Date()
    )
}
