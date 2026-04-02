import SwiftUI

struct BooleanView: View {
    let request: APIRequest
    let config: BooleanConfig
    var onSubmit: (Bool) -> Void = { _ in }

    @State private var selectedValue: Bool?

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: HSpacing.sectionGap) {
                    RequestContentView(request: request)
                        .padding(.horizontal, HSpacing.screenHorizontal)
                        .padding(.top, HSpacing.sectionGap)

                    VStack(spacing: HSpacing.elementGap) {
                        booleanOption(label: config.resolvedTrueLabel, value: true)
                        booleanOption(label: config.resolvedFalseLabel, value: false)
                    }
                    .padding(.horizontal, HSpacing.screenHorizontal)
                }
            }

            VStack {
                Divider()
                Button {
                    if let value = selectedValue {
                        onSubmit(value)
                    }
                } label: {
                    Label("Submit", systemImage: "paperplane.fill")
                }
                .buttonStyle(PrimaryButtonStyle(isEnabled: selectedValue != nil))
                .disabled(selectedValue == nil)
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

    private func booleanOption(label: String, value: Bool) -> some View {
        let isSelected = selectedValue == value

        return Button {
            withAnimation(.snappy(duration: 0.2)) {
                selectedValue = value
            }
        } label: {
            HStack {
                Text(label)
                    .font(.body)
                    .foregroundStyle(
                        isSelected ? Color.bgColor : Color.labelColor
                    )

                Spacer()

                if isSelected {
                    Image(systemName: "checkmark")
                        .font(.body.bold())
                        .foregroundStyle(Color.bgColor)
                }
            }
            .padding(.horizontal, 20)
            .frame(height: HSize.optionHeight)
            .background(
                isSelected ? Color.labelColor : Color.secondaryBgColor
            )
            .clipShape(RoundedRectangle(cornerRadius: HRadius.option))
        }
        .buttonStyle(.plain)
    }
}

#Preview {
    NavigationStack {
        BooleanView(
            request: MockData.timedOutRequest,
            config: try! ResponseConfigParser.parse(
                BooleanConfig.self,
                from: MockData.booleanConfigJSON
            )
        )
    }
}
