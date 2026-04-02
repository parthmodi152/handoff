//
//  MediumWidgetView.swift
//  HandoffWidgetExtension
//
//  systemMedium widget: pending count header + up to 3 tappable request rows.
//

import HandoffKit
import SwiftUI
import WidgetKit

struct MediumWidgetView: View {
    let data: WidgetData
    var loopName: String?

    var body: some View {
        if data.totalPending == 0 {
            VStack(spacing: 8) {
                Image(systemName: "checkmark.circle.fill")
                    .font(.system(size: 36))
                    .foregroundStyle(.green)
                Text("All caught up")
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(.secondary)
                if let loopName {
                    Text(loopName)
                        .font(.caption2)
                        .foregroundStyle(.tertiary)
                }
            }
            .frame(maxWidth: .infinity)
            .containerBackground(.fill.tertiary, for: .widget)
            .widgetURL(URL(string: "handoff://home"))
        } else {
            VStack(alignment: .leading, spacing: 6) {
                // Header
                HStack {
                    HStack(spacing: 4) {
                        Image(systemName: "hourglass")
                            .font(.caption2)
                        Text("\(data.totalPending) Pending")
                            .font(.caption.weight(.semibold))
                    }
                    .foregroundStyle(.orange)

                    Spacer()

                    if let displayName = loopName ?? data.topLoopName {
                        if let loopName {
                            Text(loopName)
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                                .lineLimit(1)
                        } else {
                            Text("\(displayName) (\(data.topLoopPendingCount))")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                                .lineLimit(1)
                        }
                    }
                }

                // Request rows (up to 3) — each is a deep link
                ForEach(Array(data.topRequests.prefix(3))) { req in
                    Link(destination: URL(string: "handoff://request/\(req.id)")!) {
                        HStack(spacing: 6) {
                            // Priority indicator with correct values
                            Circle()
                                .fill(priorityColor(req.priority))
                                .frame(width: 6, height: 6)

                            Text(req.text)
                                .font(.caption)
                                .lineLimit(1)
                                .foregroundStyle(.primary)

                            Spacer()

                            if let timeoutAt = req.timeoutAt, timeoutAt > Date() {
                                Text(timeoutAt, style: .timer)
                                    .monospacedDigit()
                                    .font(.caption2)
                                    .foregroundStyle(.orange)
                                    .frame(width: 44, alignment: .trailing)
                            }
                        }
                    }
                }

                // Overflow
                if data.totalPending > 3 {
                    Text("+ \(data.totalPending - 3) more")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
            .containerBackground(.fill.tertiary, for: .widget)
        }
    }

    private func priorityColor(_ priority: String) -> Color {
        switch priority {
        case "critical": return .red
        case "high": return .orange
        case "medium": return .yellow
        default: return .secondary.opacity(0.4)
        }
    }
}
