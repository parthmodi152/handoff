import SwiftUI

struct StatusBadge: View {
    let status: String

    private var label: String {
        switch status {
        case "completed": return "Completed"
        case "timeout": return "Timed Out"
        case "cancelled": return "Cancelled"
        default: return status.capitalized
        }
    }

    private var color: Color {
        switch status {
        case "completed": return .green
        case "timeout": return .orange
        case "cancelled": return .red
        default: return .secondary
        }
    }

    var body: some View {
        Text(label)
            .font(.caption2.bold())
            .foregroundStyle(color)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(color.opacity(0.12))
            .clipShape(RoundedRectangle(cornerRadius: HRadius.statusBadge))
    }
}

#Preview {
    HStack {
        StatusBadge(status: "completed")
        StatusBadge(status: "timeout")
        StatusBadge(status: "cancelled")
    }
    .padding()
}
