//
//  SmallWidgetView.swift
//  HandoffWidgetExtension
//
//  systemSmall widget: pending count + top request preview.
//

import HandoffKit
import SwiftUI
import WidgetKit

struct SmallWidgetView: View {
    let data: WidgetData
    var loopName: String?

    var body: some View {
        if data.totalPending == 0 {
            // All caught up
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
            .containerBackground(.fill.tertiary, for: .widget)
            .widgetURL(URL(string: "handoff://home"))
        } else {
            VStack(spacing: 4) {
                HStack {
                    Image(systemName: "hourglass")
                        .font(.caption)
                        .foregroundStyle(.orange)
                    Text("Pending")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(.secondary)
                    Spacer()
                }

                Spacer()

                Text("\(data.totalPending)")
                    .font(.system(size: 48, weight: .bold, design: .rounded))
                    .foregroundStyle(.primary)
                    .contentTransition(.numericText())

                Spacer()

                // Show the top request text for quick context
                if let first = data.topRequests.first {
                    Text(first.text)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                        .lineLimit(2)
                } else if let displayName = loopName ?? data.topLoopName {
                    HStack(spacing: 4) {
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .font(.system(size: 8))
                        Text(displayName)
                            .lineLimit(1)
                    }
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                }
            }
            .containerBackground(.fill.tertiary, for: .widget)
            // Deep link to the first request if available
            .widgetURL(
                data.topRequests.first.map { URL(string: "handoff://request/\($0.id)") } ?? URL(string: "handoff://home")
            )
        }
    }
}
