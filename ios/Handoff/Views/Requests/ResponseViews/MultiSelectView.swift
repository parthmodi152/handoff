import SwiftUI

struct MultiSelectView: View {
    let request: APIRequest
    let config: MultiSelectConfig
    var onSubmit: ([String]) -> Void = { _ in }

    @State private var selectedOptions: Set<String> = []

    private var canSubmit: Bool {
        let count = selectedOptions.count
        let min = config.minSelections ?? 1
        let max = config.maxSelections ?? config.options.count
        return count >= min && count <= max
    }

    private var selectionHint: String? {
        if let min = config.minSelections, let max = config.maxSelections {
            return "Select \(min)–\(max) options"
        } else if let min = config.minSelections {
            return "Select at least \(min)"
        } else if let max = config.maxSelections {
            return "Select up to \(max)"
        }
        return nil
    }

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: HSpacing.sectionGap) {
                    VStack(alignment: .leading, spacing: 8) {
                        RequestContentView(request: request)

                        if let hint = selectionHint {
                            Text(hint)
                                .font(.caption)
                                .foregroundStyle(Color.secondaryLabelColor)
                        }
                    }
                    .padding(.horizontal, HSpacing.screenHorizontal)
                    .padding(.top, HSpacing.sectionGap)

                    VStack(spacing: HSpacing.elementGap) {
                        ForEach(config.options) { option in
                            let isSelected = selectedOptions.contains(option.value)

                            Button {
                                withAnimation(.snappy(duration: 0.2)) {
                                    if isSelected {
                                        selectedOptions.remove(option.value)
                                    } else {
                                        selectedOptions.insert(option.value)
                                    }
                                }
                            } label: {
                                HStack {
                                    Text(option.label)
                                        .font(.body)
                                        .foregroundStyle(
                                            isSelected
                                                ? Color.bgColor
                                                : Color.labelColor
                                        )

                                    Spacer()

                                    Image(systemName: isSelected ? "checkmark.square.fill" : "square")
                                        .font(.body)
                                        .foregroundStyle(
                                            isSelected
                                                ? Color.bgColor
                                                : Color.tertiaryLabelColor
                                        )
                                }
                                .padding(.horizontal, 20)
                                .frame(height: HSize.optionHeight)
                                .background(
                                    isSelected
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

            VStack {
                Divider()
                Button {
                    onSubmit(Array(selectedOptions))
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
    }
}

#Preview {
    NavigationStack {
        MultiSelectView(
            request: MockData.multiSelectRequest,
            config: try! ResponseConfigParser.parse(
                MultiSelectConfig.self,
                from: MockData.multiSelectConfigJSON
            )
        )
    }
}
