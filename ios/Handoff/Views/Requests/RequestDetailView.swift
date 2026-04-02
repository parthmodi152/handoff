import SwiftUI

struct RequestDetailView: View {
    let request: APIRequest
    @Environment(RequestsManager.self) private var requestsManager
    @Environment(\.dismiss) private var dismiss
    @State private var isSubmitting = false
    @State private var errorMessage: String?

    var body: some View {
        Group {
            if request.isPending {
                pendingResponseView
            } else {
                completedDetailView
            }
        }
        .constrainedWidth()
        .frame(maxWidth: .infinity)
        .inlineNavigationTitle()
        .overlay {
            if isSubmitting {
                Color.bgColor.opacity(0.6)
                    .overlay { ProgressView() }
                    .ignoresSafeArea()
            }
        }
        .alert("Error", isPresented: .init(
            get: { errorMessage != nil },
            set: { if !$0 { errorMessage = nil } }
        )) {
            Button("OK") { errorMessage = nil }
        } message: {
            Text(errorMessage ?? "")
        }
    }

    // MARK: - Submit Response

    private func submitResponse(_ data: Any) {
        guard !isSubmitting else { return }
        isSubmitting = true
        Task {
            do {
                try await requestsManager.respond(requestId: request.id, responseData: data)
                dismiss()
            } catch {
                errorMessage = error.localizedDescription
            }
            isSubmitting = false
        }
    }

    // MARK: - Pending: Show appropriate response input

    @ViewBuilder
    private var pendingResponseView: some View {
        switch request.responseType {
        case "single_select":
            if let config = try? ResponseConfigParser.parse(
                SingleSelectConfig.self,
                from: request.responseConfigJSON
            ) {
                SingleSelectView(request: request, config: config) { selectedValue in
                    let option = config.options.first { $0.value == selectedValue }
                    submitResponse([
                        "selected_value": selectedValue,
                        "selected_label": option?.label ?? selectedValue
                    ])
                }
            }

        case "multi_select":
            if let config = try? ResponseConfigParser.parse(
                MultiSelectConfig.self,
                from: request.responseConfigJSON
            ) {
                MultiSelectView(request: request, config: config) { selectedValues in
                    let labels = selectedValues.compactMap { val in
                        config.options.first { $0.value == val }?.label ?? val
                    }
                    submitResponse([
                        "selected_values": selectedValues,
                        "selected_labels": labels
                    ])
                }
            }

        case "boolean":
            if let config = try? ResponseConfigParser.parse(
                BooleanConfig.self,
                from: request.responseConfigJSON
            ) {
                BooleanView(request: request, config: config) { value in
                    submitResponse([
                        "boolean": value,
                        "boolean_label": value ? config.resolvedTrueLabel : config.resolvedFalseLabel
                    ])
                }
            }

        case "rating":
            if let config = try? ResponseConfigParser.parse(
                RatingConfig.self,
                from: request.responseConfigJSON
            ) {
                RatingView(request: request, config: config) { rating in
                    var data: [String: Any] = ["rating": rating]
                    if let label = config.labels?[String(rating)] {
                        data["rating_label"] = label
                    }
                    submitResponse(data)
                }
            }

        case "text":
            if let config = try? ResponseConfigParser.parse(
                TextResponseConfig.self,
                from: request.responseConfigJSON
            ) {
                TextResponseView(request: request, config: config) { text in
                    submitResponse(text)
                }
            }

        case "number":
            if let config = try? ResponseConfigParser.parse(
                NumberConfig.self,
                from: request.responseConfigJSON
            ) {
                NumberResponseView(request: request, config: config) { number in
                    submitResponse([
                        "number": number,
                        "formatted_value": String(number)
                    ])
                }
            }

        default:
            ContentUnavailableView(
                "Unsupported Type",
                systemImage: "questionmark.circle",
                description: Text("Response type \"\(request.responseType)\" is not supported.")
            )
        }
    }

    // MARK: - Completed: Show request details and response summary

    private var completedDetailView: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: HSpacing.sectionGap) {
                // Header
                VStack(alignment: .leading, spacing: 8) {
                    HStack {
                        Text(request.loopNameDisplay)
                            .font(.caption)
                            .foregroundStyle(Color.secondaryLabelColor)

                        Spacer()

                        StatusBadge(status: request.status)
                    }

                    Text(request.title)
                        .font(.headline)
                        .foregroundStyle(Color.labelColor)

                    RequestContentView(request: request)
                }

                Divider()

                // Details section
                VStack(alignment: .leading, spacing: 12) {
                    Text("Details")
                        .font(.headline)

                    detailRow(label: "Priority", value: request.priority.capitalized)

                    if let responseTime = request.responseTimeSeconds {
                        detailRow(label: "Response Time", value: "\(Int(responseTime))s")
                    }

                    if let responseAt = request.responseAtDate {
                        detailRow(
                            label: "Responded At",
                            value: responseAt.formatted(date: .abbreviated, time: .shortened)
                        )
                    }
                }

                if let responseJSON = request.responseDataJSON {
                    Divider()

                    VStack(alignment: .leading, spacing: 8) {
                        Text("Response")
                            .font(.headline)

                        Text(responseJSON)
                            .font(.subheadline.monospaced())
                            .foregroundStyle(Color.secondaryLabelColor)
                            .padding(12)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(Color.secondaryBgColor)
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                    }
                }
            }
            .padding(HSpacing.screenHorizontal)
        }
        .navigationTitle("Request")
    }

    private func detailRow(label: String, value: String) -> some View {
        HStack {
            Text(label)
                .font(.subheadline)
                .foregroundStyle(Color.secondaryLabelColor)

            Spacer()

            Text(value)
                .font(.subheadline)
                .foregroundStyle(Color.labelColor)
        }
    }
}

#Preview("Pending - Single Select") {
    NavigationStack {
        RequestDetailView(request: MockData.singleSelectRequest)
    }
    .environment(MockData.previewRequestsManager())
}

#Preview("Pending - Boolean") {
    NavigationStack {
        RequestDetailView(request: MockData.booleanRequest)
    }
    .environment(MockData.previewRequestsManager())
}

#Preview("Completed") {
    NavigationStack {
        RequestDetailView(request: MockData.completedRequest)
    }
    .environment(MockData.previewRequestsManager())
}
