import SwiftUI

struct RequestCard: View {
    let request: APIRequest
    var showLoopName: Bool = true

    private var priorityColor: Color {
        switch request.priority {
        case "critical": return .red
        case "high": return .orange
        case "medium": return .yellow
        default: return .clear
        }
    }

    private var showPriorityBar: Bool {
        request.priority == "critical" || request.priority == "high"
    }

    var body: some View {
        HStack(spacing: 0) {
            // Thin priority accent bar
            if showPriorityBar {
                RoundedRectangle(cornerRadius: 2)
                    .fill(priorityColor)
                    .frame(width: 3)
                    .padding(.trailing, 10)
            }

            VStack(alignment: .leading, spacing: 4) {
                // Top line: loop name + relative time
                HStack(alignment: .firstTextBaseline) {
                    if showLoopName {
                        Text(request.loopNameDisplay)
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }

                    Spacer()

                    // Auto-updating relative time (Apple pattern)
                    Text(request.createdAtDate, style: .relative)
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                }

                // Title — the primary content
                Text(request.title)
                    .font(.body)
                    .foregroundStyle(.primary)
                    .lineLimit(2)

                // Bottom line: countdown timer + priority badge
                HStack(spacing: 6) {
                    if let timeout = request.timeoutDate, timeout > Date.now {
                        // Auto-updating countdown timer (Apple pattern)
                        HStack(spacing: 4) {
                            Image(systemName: "timer")
                                .font(.caption2)
                            Text(timeout, style: .relative)
                                .font(.caption)
                        }
                        .foregroundStyle(.orange)
                    }

                    Spacer()

                    if request.priority == "critical" {
                        Text("Critical")
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(.red, in: Capsule())
                    } else if request.priority == "high" {
                        Text("High")
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(.orange)
                    }
                }
            }
        }
        .padding(.vertical, 4)
    }
}

#Preview {
    List {
        RequestCard(request: MockData.booleanRequest)
        RequestCard(request: MockData.singleSelectRequest)
        RequestCard(request: MockData.multiSelectRequest)
        RequestCard(request: MockData.numberRequest)
        RequestCard(request: MockData.ratingRequest)
        RequestCard(request: MockData.textRequest)
    }
}
