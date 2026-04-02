import SwiftUI

struct TextResponseView: View {
    let request: APIRequest
    let config: TextResponseConfig
    var onSubmit: (String) -> Void = { _ in }

    @State private var text: String = ""
    @FocusState private var isTextFocused: Bool

    private var canSubmit: Bool {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        let minLen = config.minLength ?? 1
        let maxLen = config.maxLength ?? Int.max
        return trimmed.count >= minLen && trimmed.count <= maxLen
    }

    private var characterCount: String? {
        guard let maxLength = config.maxLength else { return nil }
        return "\(text.count)/\(maxLength)"
    }

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: HSpacing.sectionGap) {
                    RequestContentView(request: request)
                        .padding(.top, HSpacing.sectionGap)

                    VStack(alignment: .trailing, spacing: 8) {
                        TextEditor(text: $text)
                            .font(.body)
                            .frame(minHeight: 150)
                            .scrollContentBackground(.hidden)
                            .padding(12)
                            .background(Color.secondaryBgColor)
                            .clipShape(RoundedRectangle(cornerRadius: HRadius.option))
                            .overlay(
                                RoundedRectangle(cornerRadius: HRadius.option)
                                    .stroke(
                                        isTextFocused ? Color.labelColor : Color.separatorColor,
                                        lineWidth: isTextFocused ? 2 : 1
                                    )
                            )
                            .focused($isTextFocused)
                            .overlay(alignment: .topLeading) {
                                if text.isEmpty, let placeholder = config.placeholder {
                                    Text(placeholder)
                                        .font(.body)
                                        .foregroundStyle(Color.placeholderColor)
                                        .padding(16)
                                        .allowsHitTesting(false)
                                }
                            }

                        if let count = characterCount {
                            Text(count)
                                .font(.caption)
                                .foregroundStyle(Color.secondaryLabelColor)
                        }
                    }
                }
                .padding(.horizontal, HSpacing.screenHorizontal)
            }

            VStack {
                Divider()
                Button {
                    onSubmit(text.trimmingCharacters(in: .whitespacesAndNewlines))
                } label: {
                    Label("Submit", systemImage: "paperplane.fill")
                }
                .buttonStyle(PrimaryButtonStyle(isEnabled: canSubmit))
                .disabled(!canSubmit)
                .padding(.horizontal, HSpacing.screenHorizontal)
                .padding(.vertical, 12)
            }
            .background(Color.bgColor)
        }
        .constrainedWidth()
        .frame(maxWidth: .infinity)
        .navigationTitle(request.title)
        .inlineNavigationTitle()
        .toolbar {
            ToolbarItemGroup(placement: .keyboard) {
                Spacer()
                Button("Done") {
                    isTextFocused = false
                }
            }
        }
    }
}

#Preview {
    NavigationStack {
        TextResponseView(
            request: MockData.textRequest,
            config: try! ResponseConfigParser.parse(
                TextResponseConfig.self,
                from: MockData.textConfigJSON
            )
        )
    }
}
