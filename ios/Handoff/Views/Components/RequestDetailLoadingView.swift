import SwiftUI

/// Fetches a request by ID and displays its detail view.
/// Used when deep-linking to a request that hasn't been loaded into memory yet.
struct RequestDetailLoadingView: View {
    let requestId: String
    @Environment(RequestsManager.self) private var requestsManager
    @State private var request: APIRequest?
    @State private var errorMessage: String?

    var body: some View {
        Group {
            if let request {
                RequestDetailView(request: request)
            } else if let errorMessage {
                ContentUnavailableView(
                    "Request Not Found",
                    systemImage: "exclamationmark.triangle",
                    description: Text(errorMessage)
                )
            } else {
                ProgressView("Loading request...")
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .task {
            do {
                request = try await requestsManager.getRequest(id: requestId)
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }
}
