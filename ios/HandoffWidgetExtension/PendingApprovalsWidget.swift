//
//  PendingApprovalsWidget.swift
//  HandoffWidgetExtension
//
//  Widget definition for Pending Approvals.
//  Uses AppIntentConfiguration for user-configurable loop selection.
//

import AppIntents
import HandoffKit
import SwiftUI
import WidgetKit

struct PendingApprovalsWidget: Widget {
    let kind = "PendingApprovals"

    var body: some WidgetConfiguration {
        AppIntentConfiguration(
            kind: kind,
            intent: SelectLoopIntent.self,
            provider: PendingApprovalsProvider()
        ) { entry in
            PendingApprovalsEntryView(entry: entry)
        }
        .configurationDisplayName("Pending Approvals")
        .description("Shows your pending approval requests.")
        .supportedFamilies([
            .systemSmall, .systemMedium,
            .accessoryCircular, .accessoryRectangular, .accessoryInline
        ])
    }
}

// MARK: - Entry View Router

struct PendingApprovalsEntryView: View {
    @Environment(\.widgetFamily) var family
    let entry: PendingApprovalsEntry

    var body: some View {
        switch family {
        case .systemSmall:
            SmallWidgetView(data: entry.data, loopName: entry.selectedLoopName)
        case .systemMedium:
            MediumWidgetView(data: entry.data, loopName: entry.selectedLoopName)
        case .accessoryCircular:
            CircularWidgetView(data: entry.data)
        case .accessoryRectangular:
            RectangularWidgetView(data: entry.data, loopName: entry.selectedLoopName)
        case .accessoryInline:
            InlineWidgetView(data: entry.data)
        default:
            SmallWidgetView(data: entry.data, loopName: entry.selectedLoopName)
        }
    }
}

// MARK: - Previews

#Preview("Small", as: .systemSmall) {
    PendingApprovalsWidget()
} timeline: {
    PendingApprovalsEntry(date: Date(), data: .preview, selectedLoopName: nil)
    PendingApprovalsEntry(date: Date(), data: .preview, selectedLoopName: "DevOps")
    PendingApprovalsEntry(date: Date(), data: .empty, selectedLoopName: nil)
}

#Preview("Medium", as: .systemMedium) {
    PendingApprovalsWidget()
} timeline: {
    PendingApprovalsEntry(date: Date(), data: .preview, selectedLoopName: nil)
    PendingApprovalsEntry(date: Date(), data: .preview, selectedLoopName: "DevOps")
    PendingApprovalsEntry(date: Date(), data: .empty, selectedLoopName: nil)
}

#Preview("Circular", as: .accessoryCircular) {
    PendingApprovalsWidget()
} timeline: {
    PendingApprovalsEntry(date: Date(), data: .preview, selectedLoopName: nil)
    PendingApprovalsEntry(date: Date(), data: .empty, selectedLoopName: nil)
}

#Preview("Rectangular", as: .accessoryRectangular) {
    PendingApprovalsWidget()
} timeline: {
    PendingApprovalsEntry(date: Date(), data: .preview, selectedLoopName: nil)
    PendingApprovalsEntry(date: Date(), data: .empty, selectedLoopName: nil)
}

#Preview("Inline", as: .accessoryInline) {
    PendingApprovalsWidget()
} timeline: {
    PendingApprovalsEntry(date: Date(), data: .preview, selectedLoopName: nil)
    PendingApprovalsEntry(date: Date(), data: .empty, selectedLoopName: nil)
}
