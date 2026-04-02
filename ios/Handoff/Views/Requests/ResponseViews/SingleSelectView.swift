import SwiftUI

struct SingleSelectView: View {
    let request: APIRequest
    let config: SingleSelectConfig
    var onSubmit: (String) -> Void = { _ in }

    @State private var selectedOption: String?

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: HSpacing.sectionGap) {
                    RequestContentView(request: request)
                        .padding(.horizontal, HSpacing.screenHorizontal)
                        .padding(.top, HSpacing.sectionGap)

                    VStack(spacing: HSpacing.elementGap) {
                        ForEach(config.options) { option in
                            Button {
                                withAnimation(.snappy(duration: 0.2)) {
                                    selectedOption = option.value
                                }
                            } label: {
                                HStack {
                                    Text(option.label)
                                        .font(.body)
                                        .foregroundStyle(
                                            selectedOption == option.value
                                                ? Color.bgColor
                                                : Color.labelColor
                                        )

                                    Spacer()

                                    if selectedOption == option.value {
                                        Image(systemName: "checkmark")
                                            .font(.body.bold())
                                            .foregroundStyle(Color.bgColor)
                                    }
                                }
                                .padding(.horizontal, 20)
                                .frame(height: HSize.optionHeight)
                                .background(
                                    selectedOption == option.value
                                        ? Color.labelColor
                                        : Color.secondaryBgColor
                                )
                                .clipShape(RoundedRectangle(cornerRadius: HRadius.option))
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(.horizontal, HSpacing.screenHorizontal)
                }
            }

            // Submit button
            VStack {
                Divider()
                Button {
                    if let selected = selectedOption {
                        onSubmit(selected)
                    }
                } label: {
                    Label("Submit", systemImage: "paperplane.fill")
                }
                .buttonStyle(PrimaryButtonStyle(isEnabled: selectedOption != nil))
                .disabled(selectedOption == nil)
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
        SingleSelectView(
            request: MockData.singleSelectRequest,
            config: try! ResponseConfigParser.parse(
                SingleSelectConfig.self,
                from: MockData.singleSelectConfigJSON
            )
        )
    }
}
