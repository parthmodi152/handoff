//
//  SelectLoopIntent.swift
//  HandoffWidgetExtension
//
//  App Intent for configuring which loop the widget tracks.
//

import AppIntents
import HandoffKit
import OSLog
import WidgetKit

private let logger = Logger(subsystem: "sh.hitl.handoff.widget", category: "LoopEntityQuery")

// MARK: - Loop Entity

struct LoopEntity: AppEntity {
    static var typeDisplayRepresentation = TypeDisplayRepresentation(name: "Loop")
    static var defaultQuery = LoopEntityQuery()

    var id: String
    var name: String

    var displayRepresentation: DisplayRepresentation {
        DisplayRepresentation(title: "\(name)")
    }
}

// MARK: - Loop Entity Query

struct LoopEntityQuery: EntityStringQuery {
    func entities(for identifiers: [String]) async throws -> [LoopEntity] {
        logger.info("entities(for:) called with \(identifiers.count) identifiers: \(identifiers.joined(separator: ", "))")
        let loops = WidgetDataStore.readLoops()
        logger.info("entities(for:) readLoops returned \(loops.count) loops")
        return identifiers.compactMap { id in
            guard let loop = loops.first(where: { $0.id == id }) else { return nil }
            return LoopEntity(id: loop.id, name: loop.name)
        }
    }

    func entities(matching string: String) async throws -> [LoopEntity] {
        logger.info("entities(matching:) called with string: '\(string)'")
        let loops = WidgetDataStore.readLoops()
        logger.info("entities(matching:) readLoops returned \(loops.count) loops")
        if string.isEmpty {
            let result = loops.map { LoopEntity(id: $0.id, name: $0.name) }
            logger.info("entities(matching:) returning \(result.count) entities (empty string)")
            return result
        }
        let filtered = loops
            .filter { $0.name.localizedCaseInsensitiveContains(string) }
            .map { LoopEntity(id: $0.id, name: $0.name) }
        logger.info("entities(matching:) returning \(filtered.count) filtered entities")
        return filtered
    }

    func suggestedEntities() async throws -> [LoopEntity] {
        let loops = WidgetDataStore.readLoops()
        logger.info("suggestedEntities() readLoops returned \(loops.count) loops")
        return loops.map { LoopEntity(id: $0.id, name: $0.name) }
    }

    func defaultResult() async -> LoopEntity? {
        logger.info("defaultResult() called")
        return nil
    }
}

// MARK: - Widget Configuration Intent

struct SelectLoopIntent: WidgetConfigurationIntent {
    static var title: LocalizedStringResource = "Select Loop"
    static var description: IntentDescription = "Choose which loop to show pending approvals for."

    @Parameter(title: "Loop")
    var loop: LoopEntity?
}
