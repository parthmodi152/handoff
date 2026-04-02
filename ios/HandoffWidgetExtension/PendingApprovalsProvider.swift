//
//  PendingApprovalsProvider.swift
//  HandoffWidgetExtension
//
//  Timeline provider for the Pending Approvals widget.
//  Reads data from shared UserDefaults written by the main app.
//  Supports user-configurable loop selection via SelectLoopIntent.
//

import AppIntents
import HandoffKit
import WidgetKit

struct PendingApprovalsEntry: TimelineEntry {
    let date: Date
    let data: WidgetData
    let selectedLoopName: String?
}

struct PendingApprovalsProvider: AppIntentTimelineProvider {
    func placeholder(in context: Context) -> PendingApprovalsEntry {
        PendingApprovalsEntry(date: Date(), data: .preview, selectedLoopName: nil)
    }

    func snapshot(for configuration: SelectLoopIntent, in context: Context) async -> PendingApprovalsEntry {
        if context.isPreview {
            return PendingApprovalsEntry(date: Date(), data: .preview, selectedLoopName: nil)
        }
        let data = filteredData(for: configuration)
        return PendingApprovalsEntry(date: Date(), data: data, selectedLoopName: configuration.loop?.name)
    }

    func timeline(for configuration: SelectLoopIntent, in context: Context) async -> Timeline<PendingApprovalsEntry> {
        let data = filteredData(for: configuration)
        let entry = PendingApprovalsEntry(date: Date(), data: data, selectedLoopName: configuration.loop?.name)
        // .never — the main app controls reloads via WidgetCenter
        return Timeline(entries: [entry], policy: .never)
    }

    private func filteredData(for configuration: SelectLoopIntent) -> WidgetData {
        let allData = WidgetDataStore.read()
        guard let loopId = configuration.loop?.id else {
            // No loop selected — show all pending
            return allData
        }
        return allData.filtered(byLoopId: loopId)
    }
}
