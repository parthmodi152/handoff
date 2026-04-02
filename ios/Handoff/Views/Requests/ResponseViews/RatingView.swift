import SwiftUI

struct RatingView: View {
    let request: APIRequest
    let config: RatingConfig
    var onSubmit: (Int) -> Void = { _ in }

    @State private var selectedRating: Int = 0

    private var maxStars: Int {
        config.maxValue
    }

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(spacing: 32) {
                    RequestContentView(request: request)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.top, HSpacing.sectionGap)

                    // Star rating
                    VStack(spacing: 16) {
                        HStack(spacing: 12) {
                            ForEach(1...maxStars, id: \.self) { star in
                                Button {
                                    withAnimation(.snappy(duration: 0.15)) {
                                        selectedRating = star
                                    }
                                } label: {
                                    Image(systemName: star <= selectedRating ? "star.fill" : "star")
                                        .font(.system(size: HSize.starSize))
                                        .foregroundStyle(
                                            star <= selectedRating
                                                ? Color.brandGold
                                                : Color.tertiaryLabelColor
                                        )
                                        .symbolEffect(.bounce, value: selectedRating == star)
                                }
                                .buttonStyle(.plain)
                            }
                        }

                        // Labels
                        if config.labels != nil {
                            HStack {
                                if let minLabel = config.minLabel {
                                    Text(minLabel)
                                        .font(.caption)
                                        .foregroundStyle(Color.secondaryLabelColor)
                                }
                                Spacer()
                                if let maxLabel = config.maxLabel {
                                    Text(maxLabel)
                                        .font(.caption)
                                        .foregroundStyle(Color.secondaryLabelColor)
                                }
                            }
                        }

                        if selectedRating > 0 {
                            Text("\(selectedRating) of \(maxStars)")
                                .font(.headline)
                                .foregroundStyle(Color.brandGold)
                        }
                    }
                }
                .padding(.horizontal, HSpacing.screenHorizontal)
            }

            VStack {
                Divider()
                Button {
                    onSubmit(selectedRating)
                } label: {
                    Label("Submit", systemImage: "paperplane.fill")
                }
                .buttonStyle(PrimaryButtonStyle(isEnabled: selectedRating > 0))
                .disabled(selectedRating == 0)
                .padding(.horizontal, HSpacing.screenHorizontal)
                .padding(.vertical, 12)
            }
            .background(Color.bgColor)
        }
        .constrainedWidth()
        .frame(maxWidth: .infinity)
        .navigationTitle(request.title)
        .inlineNavigationTitle()
    }
}

#Preview {
    NavigationStack {
        RatingView(
            request: MockData.ratingRequest,
            config: try! ResponseConfigParser.parse(
                RatingConfig.self,
                from: MockData.ratingConfigJSON
            )
        )
    }
}
