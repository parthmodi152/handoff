import SwiftUI

struct HomeView: View {
    @Binding var showJoinLoop: Bool
    @Binding var selectedRequest: APIRequest?
    @Environment(RequestsManager.self) private var requestsManager
    @Environment(NavigationManager.self) private var navigationManager

    @State private var swipeSubmittingId: String?

    /// Unique loops from pending requests, ordered by first appearance
    private var availableLoops: [(id: String, name: String)] {
        var seen = Set<String>()
        var result: [(id: String, name: String)] = []
        for request in requestsManager.pendingRequests {
            if seen.insert(request.loopId).inserted {
                result.append((id: request.loopId, name: request.loopNameDisplay))
            }
        }
        return result
    }

    /// Requests filtered by selected loop
    private var filteredRequests: [APIRequest] {
        guard let loopId = navigationManager.selectedLoopId else {
            return requestsManager.pendingRequests
        }
        return requestsManager.pendingRequests.filter { $0.loopId == loopId }
    }

    /// Group filtered requests by loop (includes loopId for Live Activity management)
    private var groupedRequests: [(loopId: String, loopName: String, requests: [APIRequest])] {
        var groups: [(loopId: String, loopName: String, requests: [APIRequest])] = []
        var seen = Set<String>()
        for request in filteredRequests {
            if !seen.contains(request.loopId) {
                seen.insert(request.loopId)
                let loopRequests = filteredRequests.filter { $0.loopId == request.loopId }
                groups.append((loopId: request.loopId, loopName: request.loopNameDisplay, requests: loopRequests))
            }
        }
        return groups
    }

    var body: some View {
        Group {
            if requestsManager.isLoadingPending && requestsManager.pendingRequests.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if requestsManager.pendingRequests.isEmpty {
                ContentUnavailableView(
                    "No Pending Requests",
                    systemImage: "tray",
                    description: Text("Join a loop to start receiving requests.")
                )
            } else {
                List {
                    if availableLoops.count > 1 {
                        loopFilterRow
                    }

                    ForEach(groupedRequests, id: \.loopId) { group in
                        Section {
                            ForEach(group.requests) { request in
                                Button {
                                    selectedRequest = request
                                } label: {
                                    RequestCard(request: request)
                                }
                                .buttonStyle(.plain)
                                .swipeActions(edge: .trailing, allowsFullSwipe: true) {
                                    if request.responseType == "boolean" {
                                        // Reject / false action (trailing = destructive side)
                                        let config = try? ResponseConfigParser.parse(
                                            BooleanConfig.self,
                                            from: request.responseConfigJSON
                                        )
                                        Button(role: .destructive) {
                                            submitBooleanResponse(request: request, value: false)
                                        } label: {
                                            Label(config?.resolvedFalseLabel ?? "No", systemImage: "xmark")
                                        }
                                    }
                                }
                                .swipeActions(edge: .leading, allowsFullSwipe: true) {
                                    if request.responseType == "boolean" {
                                        // Approve / true action (leading = positive side)
                                        let config = try? ResponseConfigParser.parse(
                                            BooleanConfig.self,
                                            from: request.responseConfigJSON
                                        )
                                        Button {
                                            submitBooleanResponse(request: request, value: true)
                                        } label: {
                                            Label(config?.resolvedTrueLabel ?? "Yes", systemImage: "checkmark")
                                        }
                                        .tint(.green)
                                    }
                                }
                            }
                        } header: {
                            if groupedRequests.count > 1 || navigationManager.selectedLoopId == nil {
                                HStack {
                                    Image(systemName: "arrow.triangle.2.circlepath")
                                        .font(.caption2)
                                    Text(group.loopName)
                                    Spacer()
                                    Text("\(group.requests.count)")
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                    }
                }
                .listStyle(.inset)
            }
        }
        .navigationTitle("Handoff")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    showJoinLoop = true
                } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .sheet(item: $selectedRequest) { request in
            NavigationStack {
                RequestDetailView(request: request)
            }
            .presentationDetents([.medium, .large])
            .presentationDragIndicator(.visible)
            .presentationContentInteraction(.scrolls)
        }
        .refreshable {
            await requestsManager.fetchPending()
        }
        .task {
            if requestsManager.pendingRequests.isEmpty {
                await requestsManager.fetchPending()
            }
        }
        .overlay {
            if swipeSubmittingId != nil {
                Color.bgColor.opacity(0.4)
                    .overlay { ProgressView() }
                    .ignoresSafeArea()
                    .allowsHitTesting(true)
            }
        }
    }

    // MARK: - Quick Response via Swipe

    private func submitBooleanResponse(request: APIRequest, value: Bool) {
        guard swipeSubmittingId == nil else { return }
        swipeSubmittingId = request.id
        let config = try? ResponseConfigParser.parse(
            BooleanConfig.self,
            from: request.responseConfigJSON
        )
        Task {
            do {
                try await requestsManager.respond(
                    requestId: request.id,
                    responseData: [
                        "boolean": value,
                        "boolean_label": value ? (config?.resolvedTrueLabel ?? "Yes") : (config?.resolvedFalseLabel ?? "No")
                    ]
                )
            } catch {
                // Error is handled by RequestsManager
            }
            swipeSubmittingId = nil
        }
    }

    // MARK: - Loop Filter

    private var loopFilterRow: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: HSpacing.elementGap) {
                Button {
                    withAnimation { navigationManager.selectedLoopId = nil }
                } label: {
                    Text("All")
                        .font(.subheadline)
                        .padding(.horizontal, 14)
                        .padding(.vertical, 8)
                        .background(
                            navigationManager.selectedLoopId == nil
                                ? Color.labelColor
                                : Color.secondaryBgColor
                        )
                        .foregroundStyle(
                            navigationManager.selectedLoopId == nil
                                ? Color.bgColor
                                : Color.labelColor
                        )
                        .clipShape(Capsule())
                }
                .buttonStyle(.plain)

                ForEach(availableLoops, id: \.id) { loop in
                    Button {
                        withAnimation { navigationManager.selectedLoopId = loop.id }
                    } label: {
                        Text(loop.name)
                            .font(.subheadline)
                            .padding(.horizontal, 14)
                            .padding(.vertical, 8)
                            .background(
                                navigationManager.selectedLoopId == loop.id
                                    ? Color.labelColor
                                    : Color.secondaryBgColor
                            )
                            .foregroundStyle(
                                navigationManager.selectedLoopId == loop.id
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
        HomeView(showJoinLoop: .constant(false), selectedRequest: .constant(nil))
    }
    .environment(MockData.previewRequestsManager())
    .environment(NavigationManager())
}
