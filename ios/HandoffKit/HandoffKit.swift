//
//  HandoffKit.swift
//  HandoffKit
//
//  Shared types between the main app and widget extension.
//

import ActivityKit
import Foundation

/// App-wide ActivityAttributes for showing sensitive requests as a Live Activity.
///
/// No static attributes — the activity is app-wide, not per-loop.
/// ContentState holds the current set of sensitive (critical/high priority) requests.
public struct HandoffActivityAttributes: ActivityAttributes {
    public struct ContentState: Codable, Hashable {
        /// A sensitive request shown inside the Live Activity with interactive fields.
        public struct SensitiveRequest: Codable, Hashable {
            public let id: String
            public let loopName: String
            public let text: String           // truncated to 60 chars
            public let responseType: String   // "boolean", "single_select", etc.
            public let priority: String       // "critical" or "high"
            public let timeoutAt: Date?

            // Interactive fields for inline response:
            public let trueLabel: String?     // For boolean
            public let falseLabel: String?    // For boolean
            public let selectOptions: [SelectOption]?  // For single_select ≤3 options

            public init(
                id: String, loopName: String, text: String,
                responseType: String, priority: String, timeoutAt: Date?,
                trueLabel: String? = nil, falseLabel: String? = nil,
                selectOptions: [SelectOption]? = nil
            ) {
                self.id = id
                self.loopName = loopName
                self.text = String(text.prefix(60))
                self.responseType = responseType
                self.priority = priority
                self.timeoutAt = timeoutAt
                self.trueLabel = trueLabel
                self.falseLabel = falseLabel
                self.selectOptions = selectOptions
            }
        }

        public struct SelectOption: Codable, Hashable {
            public let value: String
            public let label: String

            public init(value: String, label: String) {
                self.value = value
                self.label = label
            }
        }

        public let requests: [SensitiveRequest]  // max 3 for 4KB budget
        public let totalCount: Int               // total sensitive count (may exceed 3)

        public init(requests: [SensitiveRequest], totalCount: Int) {
            self.requests = Array(requests.prefix(3))
            self.totalCount = totalCount
        }
    }

    public init() {}

    /// Preview sample for Xcode canvas previews.
    public static var preview: HandoffActivityAttributes {
        HandoffActivityAttributes()
    }
}
