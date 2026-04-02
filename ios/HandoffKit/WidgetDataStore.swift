//
//  WidgetDataStore.swift
//  HandoffKit
//
//  Read/write/clear helper for widget data via App Groups shared UserDefaults.
//  Used by the main app (write) and widget extension (read).
//

import Foundation
import OSLog
import WidgetKit

public enum WidgetDataStore {
    private static let suiteName = "group.sh.hitl.handoff"
    private static let key = "widget_pending_data"
    private static let loopsKey = "widget_available_loops"
    private static let logger = Logger(subsystem: "sh.hitl.handoff", category: "WidgetDataStore")

    // MARK: - Widget Data

    public static func write(_ data: WidgetData) {
        guard let defaults = UserDefaults(suiteName: suiteName) else { return }
        let encoded = try? JSONEncoder().encode(data)
        defaults.set(encoded, forKey: key)
    }

    public static func read() -> WidgetData {
        guard let defaults = UserDefaults(suiteName: suiteName),
              let data = defaults.data(forKey: key),
              let decoded = try? JSONDecoder().decode(WidgetData.self, from: data) else {
            return .empty
        }
        return decoded
    }

    // MARK: - Available Loops

    public static func writeLoops(_ loops: [WidgetLoopInfo]) {
        guard let defaults = UserDefaults(suiteName: suiteName) else {
            logger.error("writeLoops: failed to open UserDefaults suite '\(suiteName)'")
            return
        }
        let encoded = try? JSONEncoder().encode(loops)
        defaults.set(encoded, forKey: loopsKey)
        logger.info("writeLoops: wrote \(loops.count) loops: \(loops.map(\.name).joined(separator: ", "))")
    }

    public static func readLoops() -> [WidgetLoopInfo] {
        guard let defaults = UserDefaults(suiteName: suiteName) else {
            logger.error("readLoops: failed to open UserDefaults suite '\(suiteName)'")
            return []
        }
        let data = defaults.data(forKey: loopsKey)
        logger.info("readLoops: raw data exists = \(data != nil), bytes = \(data?.count ?? 0)")
        guard let data,
              let decoded = try? JSONDecoder().decode([WidgetLoopInfo].self, from: data) else {
            logger.warning("readLoops: decode failed or no data")
            return []
        }
        logger.info("readLoops: decoded \(decoded.count) loops: \(decoded.map(\.name).joined(separator: ", "))")
        return decoded
    }

    // MARK: - Clear & Reload

    public static func clear() {
        guard let defaults = UserDefaults(suiteName: suiteName) else { return }
        defaults.removeObject(forKey: key)
        defaults.removeObject(forKey: loopsKey)
    }

    public static func reloadWidgets() {
        WidgetCenter.shared.reloadTimelines(ofKind: "PendingApprovals")
    }
}
