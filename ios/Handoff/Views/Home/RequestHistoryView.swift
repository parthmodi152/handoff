import SwiftUI

struct RequestHistoryView: View {
    @Environment(RequestsManager.self) private var requestsManager
    @State private var selectedFilter: HistoryFilter = .all

    enum HistoryFilter: String, CaseIterable {
        case all = "All"
        case completed = "Completed"
        case timedOut = "Timed Out"
        case cancelled = "Cancelled"
    }

    private var filteredRequests: [APIRequest] {
        let requests = requestsManager.historyRequests
        switch selectedFilter {
        case .all: return requests
        case .completed: return requests.filter { $0.status == "completed" }
        case .timedOut: return requests.filter { $0.status == "timeout" }
        case .cancelled: return requests.filter { $0.status == "cancelled" }
        }
    }

    var body: some View {
        Group {
            if requestsManager.isLoadingHistory && requestsManager.historyRequests.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if requestsManager.historyRequests.isEmpty {
                ContentUnavailableView(
                    "No History",
                    systemImage: "clock.arrow.circlepath",
                    description: Text("Completed requests will appear here.")
                )
            } else {
                List {
                    filterRow

                    ForEach(filteredRequests) { request in
                        NavigationLink(value: request.id) {
                            HStack {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(request.loopNameDisplay)
                                        .font(.caption)
                                        .foregroundStyle(Color.secondaryLabelColor)

                                    Text(request.title)
                                        .font(.body)
                                        .foregroundStyle(Color.labelColor)
                                        .lineLimit(2)
                                }

                                Spacer()

                                StatusBadge(status: request.status)
                            }
                            .padding(.vertical, 4)
                        }
                    }
                }
                .listStyle(.inset)
            }
        }
        .navigationTitle("History")
        .navigationDestination(for: String.self) { requestID in
            if let request = requestsManager.historyRequests.first(where: { $0.id == requestID }) {
                RequestDetailView(request: request)
            } else {
                RequestDetailLoadingView(requestId: requestID)
            }
        }
        .refreshable {
            await requestsManager.fetchHistory()
        }
        .task {
            if requestsManager.historyRequests.isEmpty {
                await requestsManager.fetchHistory()
            }
        }
    }

    private var filterRow: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: HSpacing.elementGap) {
                ForEach(HistoryFilter.allCases, id: \.self) { filter in
                    Button {
                        withAnimation { selectedFilter = filter }
                    } label: {
                        Text(filter.rawValue)
                            .font(.subheadline)
                            .padding(.horizontal, 14)
                            .padding(.vertical, 8)
                            .background(
                                selectedFilter == filter
                                    ? Color.labelColor
                                    : Color.secondaryBgColor
                            )
                            .foregroundStyle(
                                selectedFilter == filter
                                    ? Color.bgColor
                                    : Color.labelColor
                            )
                            .clipShape(Capsule())
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, HSpacing.screenHorizontal)
        }
        .listRowInsets(EdgeInsets())
        .listRowSeparator(.hidden)
    }
}

#Preview {
    NavigationStack {
        RequestHistoryView()
    }
    .environment(MockData.previewRequestsManager())
}
