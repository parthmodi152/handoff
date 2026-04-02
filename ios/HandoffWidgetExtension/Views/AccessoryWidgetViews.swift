//
//  AccessoryWidgetViews.swift
//  HandoffWidgetExtension
//
//  Lock Screen accessory widgets: circular, rectangular, inline.
//

import HandoffKit
import SwiftUI
import WidgetKit

// MARK: - Circular (Lock Screen)

struct CircularWidgetView: View {
    let data: WidgetData

    var body: some View {
        ZStack {
            AccessoryWidgetBackground()
            if data.totalPending == 0 {
                Image(systemName: "checkmark")
                    .font(.title2.weight(.semibold))
            } else {
                Text("\(data.totalPending)")
                    .font(.title.weight(.bold))
                    .contentTransition(.numericText())
            }
        }
        .containerBackground(.clear, for: .widget)
        .widgetURL(
            data.topRequests.first.map { URL(string: "handoff://request/\($0.id)") } ?? URL(string: "handoff://home")
        )
    }
}

// MARK: - Rectangular (Lock Screen)

struct RectangularWidgetView: View {
    let data: WidgetData
    var loopName: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            if data.totalPending == 0 {
                Label("All caught up", systemImage: "checkmark.circle")
                    .font(.headline)
                if let loopName {
                    Text(loopName)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            } else {
                Label("\(data.totalPending) pending", systemImage: "hourglass")
                    .font(.headline)

                if let first = data.topRequests.first {
                    Text(first.text)
                        .font(.caption)
                        .lineLimit(1)
                        .foregroundStyle(.secondary)

                    if let displayName = loopName ?? data.topLoopName {
                        Text(displayName)
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .containerBackground(.clear, for: .widget)
        .widgetURL(
            data.topRequests.first.map { URL(string: "handoff://request/\($0.id)") } ?? URL(string: "handoff://home")
        )
    }
}

// MARK: - Inline (Lock Screen)

struct InlineWidgetView: View {
    let data: WidgetData

    var body: some View {
        if data.totalPending == 0 {
            Label("All caught up", systemImage: "checkmark.circle")
        } else {
            Label("\(data.totalPending) pending", systemImage: "hourglass")
        }
    }
}
