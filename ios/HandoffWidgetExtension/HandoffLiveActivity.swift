//
//  HandoffLiveActivity.swift
//  HandoffWidgetExtension
//
//  App-wide Live Activity showing sensitive requests with interactive buttons.
//

import ActivityKit
import AppIntents
import SwiftUI
import WidgetKit
import HandoffKit

struct HandoffLiveActivity: Widget {
    var body: some WidgetConfiguration {
        ActivityConfiguration(for: HandoffActivityAttributes.self) { context in
            // LOCK SCREEN / STANDBY presentation
            LockScreenLiveActivityView(context: context)
        } dynamicIsland: { context in
            DynamicIsland {
                // EXPANDED presentation
                DynamicIslandExpandedRegion(.leading) {
                    Image(systemName: "hand.raised.fill")
                        .font(.title3)
                        .foregroundStyle(.white)
                }
                DynamicIslandExpandedRegion(.trailing) {
                    if let nearest = nearestTimeout(context.state.requests) {
                        Text(nearest, style: .timer)
                            .monospacedDigit()
                            .font(.caption2)
                            .foregroundStyle(.orange)
                    }
                }
                DynamicIslandExpandedRegion(.center) {
                    if let first = context.state.requests.first {
                        VStack(spacing: 6) {
                            Text(first.text)
                                .lineLimit(2)
                                .font(.subheadline.weight(.medium))

                            // Interactive buttons for the top request
                            requestButtons(for: first)
                        }
                    } else {
                        Label("All clear", systemImage: "checkmark.circle.fill")
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(.green)
                    }
                }
                DynamicIslandExpandedRegion(.bottom) {
                    if context.state.totalCount > 1 {
                        Text("+ \(context.state.totalCount - 1) more")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            } compactLeading: {
                // Count badge
                Text("\(context.state.totalCount)")
                    .font(.caption.weight(.bold))
                    .foregroundStyle(hasCritical(context.state.requests) ? .orange : .primary)
            } compactTrailing: {
                if let nearest = nearestTimeout(context.state.requests) {
                    Text(nearest, style: .timer)
                        .monospacedDigit()
                        .font(.caption2)
                        .frame(width: 40)
                } else if context.state.totalCount == 0 {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(.green)
                } else {
                    Image(systemName: "circle.dotted")
                        .foregroundStyle(.secondary)
                }
            } minimal: {
                Text("\(context.state.totalCount)")
                    .font(.caption2.weight(.bold))
                    .foregroundStyle(hasCritical(context.state.requests) ? .orange : .primary)
            }
            .widgetURL(
                context.state.requests.first.map { URL(string: "handoff://request/\($0.id)")! }
                ?? URL(string: "handoff://home")!
            )
        }
        .supplementalActivityFamilies([.small])
    }
}

// MARK: - Lock Screen View

private struct LockScreenLiveActivityView: View {
    let context: ActivityViewContext<HandoffActivityAttributes>

    private var requests: [HandoffActivityAttributes.ContentState.SensitiveRequest] {
        context.state.requests
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            // Header: app name + count badge
            HStack {
                Label("Handoff", systemImage: "hand.raised.fill")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.white)

                Spacer()

                if context.state.totalCount > 0 {
                    Text("\(context.state.totalCount) sensitive")
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(.orange)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 2)
                        .background(.orange.opacity(0.15), in: Capsule())
                }
            }

            if requests.isEmpty {
                // Empty state: all clear
                HStack {
                    Spacer()
                    Label("All clear", systemImage: "checkmark.circle.fill")
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(.green)
                    Spacer()
                }
                .padding(.vertical, 4)
            } else {
                // Top request: full interactive card
                if let first = requests.first {
                    requestCard(for: first, showButtons: true)
                }

                // Remaining requests: compact rows with "Open →"
                ForEach(requests.dropFirst().prefix(2), id: \.id) { req in
                    requestCard(for: req, showButtons: false)
                }

                // "+ N more" if total exceeds displayed
                if context.state.totalCount > 3 {
                    Text("+ \(context.state.totalCount - 3) more")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .padding()
        .activityBackgroundTint(Color(red: 0.11, green: 0.11, blue: 0.12))
        .activitySystemActionForegroundColor(.white)
    }

    // MARK: - Request Card

    @ViewBuilder
    private func requestCard(for req: HandoffActivityAttributes.ContentState.SensitiveRequest, showButtons: Bool) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            // Loop name + priority
            HStack(spacing: 6) {
                Circle()
                    .fill(priorityColor(req.priority))
                    .frame(width: 6, height: 6)

                Text(req.loopName)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)

                Spacer()

                if let timeoutAt = req.timeoutAt {
                    Text(timeoutAt, style: .timer)
                        .monospacedDigit()
                        .font(.caption2)
                        .foregroundStyle(.orange)
                }
            }

            // Request text
            Text(req.text)
                .font(.caption)
                .lineLimit(2)
                .foregroundStyle(.white)

            // Buttons or "Open →"
            if showButtons {
                requestButtons(for: req)
            } else {
                Link(destination: URL(string: "handoff://request/\(req.id)")!) {
                    Text("Open →")
                        .font(.caption2.weight(.medium))
                        .foregroundStyle(.blue)
                }
            }
        }
        .padding(10)
        .background(Color.white.opacity(0.08), in: RoundedRectangle(cornerRadius: 10))
    }
}

// MARK: - Interactive Buttons

@ViewBuilder
private func requestButtons(for req: HandoffActivityAttributes.ContentState.SensitiveRequest) -> some View {
    switch req.responseType {
    case "boolean":
        HStack(spacing: 8) {
            Button(intent: RespondBooleanIntent(requestId: req.id, value: false)) {
                Text(req.falseLabel ?? "Reject")
                    .font(.caption2.weight(.semibold))
                    .frame(maxWidth: .infinity)
            }
            .tint(.red)
            .buttonStyle(.bordered)
            .controlSize(.small)

            Button(intent: RespondBooleanIntent(requestId: req.id, value: true)) {
                Text(req.trueLabel ?? "Approve")
                    .font(.caption2.weight(.semibold))
                    .frame(maxWidth: .infinity)
            }
            .tint(.green)
            .buttonStyle(.bordered)
            .controlSize(.small)
        }

    case "single_select":
        if let options = req.selectOptions, options.count <= 3 {
            HStack(spacing: 6) {
                ForEach(options, id: \.value) { option in
                    Button(intent: RespondSelectIntent(requestId: req.id, selectedValue: option.value)) {
                        Text(option.label)
                            .font(.caption2.weight(.semibold))
                            .frame(maxWidth: .infinity)
                    }
                    .tint(.blue)
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }
            }
        } else {
            // Too many options — deep link
            Link(destination: URL(string: "handoff://request/\(req.id)")!) {
                Text("Open →")
                    .font(.caption2.weight(.medium))
                    .foregroundStyle(.blue)
            }
        }

    default:
        // text, number, rating, multi_select — deep link
        Link(destination: URL(string: "handoff://request/\(req.id)")!) {
            Text("Open →")
                .font(.caption2.weight(.medium))
                .foregroundStyle(.blue)
        }
    }
}

// MARK: - Helpers

private func priorityColor(_ priority: String) -> Color {
    switch priority {
    case "critical": return .red
    case "high": return .orange
    case "medium": return .yellow
    default: return .secondary.opacity(0.4)
    }
}

private func nearestTimeout(_ requests: [HandoffActivityAttributes.ContentState.SensitiveRequest]) -> Date? {
    requests.compactMap { $0.timeoutAt }.min()
}

private func hasCritical(_ requests: [HandoffActivityAttributes.ContentState.SensitiveRequest]) -> Bool {
    requests.contains { $0.priority == "critical" }
}

// MARK: - Previews

#Preview("Live Activity", as: .content, using: HandoffActivityAttributes.preview) {
    HandoffLiveActivity()
} contentStates: {
    HandoffActivityAttributes.ContentState(
        requests: [
            .init(id: "1", loopName: "DevOps", text: "Deploy v2.4.1 to production?", responseType: "boolean", priority: "critical", timeoutAt: Date().addingTimeInterval(154), trueLabel: "Deploy", falseLabel: "Cancel"),
            .init(id: "2", loopName: "Content Review", text: "Review flagged post for policy", responseType: "single_select", priority: "high", timeoutAt: Date().addingTimeInterval(300), selectOptions: [.init(value: "approve", label: "Approve"), .init(value: "reject", label: "Reject"), .init(value: "escalate", label: "Escalate")]),
            .init(id: "3", loopName: "Refunds", text: "Provide refund justification", responseType: "text", priority: "high", timeoutAt: nil)
        ],
        totalCount: 5
    )
    HandoffActivityAttributes.ContentState(
        requests: [],
        totalCount: 0
    )
}
