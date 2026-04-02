import SwiftUI

/// Renders the request body based on `contentType`:
/// - `"markdown"` (default): renders `requestText` with inline markdown formatting
/// - `"image"`: shows the image from `imageUrl` above the markdown text
struct RequestContentView: View {
    let request: APIRequest

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if request.contentType == "image", let urlString = request.imageUrl,
               let url = URL(string: urlString) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .empty:
                        RoundedRectangle(cornerRadius: 12)
                            .fill(Color.secondaryBgColor)
                            .frame(height: 200)
                            .overlay { ProgressView() }

                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fit)
                            .clipShape(RoundedRectangle(cornerRadius: 12))

                    case .failure:
                        RoundedRectangle(cornerRadius: 12)
                            .fill(Color.secondaryBgColor)
                            .frame(height: 120)
                            .overlay {
                                Label("Image failed to load", systemImage: "photo.badge.exclamationmark")
                                    .font(.caption)
                                    .foregroundStyle(Color.secondaryLabelColor)
                            }

                    @unknown default:
                        EmptyView()
                    }
                }
            }

            markdownText
        }
    }

    private var markdownText: some View {
        Group {
            if let attributed = try? AttributedString(markdown: request.requestText,
                                                       options: .init(interpretedSyntax: .inlineOnlyPreservingWhitespace)) {
                Text(attributed)
                    .font(.body)
                    .foregroundStyle(Color.labelColor)
            } else {
                Text(request.requestText)
                    .font(.body)
                    .foregroundStyle(Color.labelColor)
            }
        }
    }
}

#Preview("Markdown") {
    RequestContentView(request: MockData.completedRequest)
        .padding()
}

#Preview("Image") {
    RequestContentView(request: MockData.imageRequest)
        .padding()
}
