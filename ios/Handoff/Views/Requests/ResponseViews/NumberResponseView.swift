import SwiftUI

struct NumberResponseView: View {
    let request: APIRequest
    let config: NumberConfig
    var onSubmit: (Double) -> Void = { _ in }

    @State private var numberText: String = ""
    @FocusState private var isFieldFocused: Bool

    private var parsedNumber: Double? {
        Double(numberText)
    }

    private var canSubmit: Bool {
        guard let value = parsedNumber else { return false }
        if let min = config.minValue, value < min { return false }
        if let max = config.maxValue, value > max { return false }
        return true
    }

    private var rangeHint: String? {
        switch (config.minValue, config.maxValue) {
        case let (min?, max?):
            return "Enter a number between \(formatted(min)) and \(formatted(max))"
        case let (min?, nil):
            return "Minimum: \(formatted(min))"
        case let (nil, max?):
            return "Maximum: \(formatted(max))"
        default:
            return nil
        }
    }

    private func formatted(_ value: Double) -> String {
        let places = config.decimalPlaces ?? 0
        if places == 0 {
            return String(Int(value))
        }
        return String(format: "%.\(places)f", value)
    }

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: HSpacing.sectionGap) {
                    RequestContentView(request: request)
                        .padding(.top, HSpacing.sectionGap)

                    VStack(alignment: .leading, spacing: 8) {
                        TextField("Enter a number", text: $numberText)
                            .font(.title.monospacedDigit())
                            #if os(iOS) || os(visionOS)
                            .keyboardType(
                                (config.decimalPlaces ?? 0) > 0
                                    ? .decimalPad
                                    : .numberPad
                            )
                            #endif
                            .multilineTextAlignment(.center)
                            .padding()
                            .background(Color.secondaryBgColor)
                            .clipShape(RoundedRectangle(cornerRadius: HRadius.option))
                            .overlay(
                                RoundedRectangle(cornerRadius: HRadius.option)
                                    .stroke(
                                        isFieldFocused ? Color.labelColor : Color.separatorColor,
                                        lineWidth: isFieldFocused ? 2 : 1
                                    )
                            )
                            .focused($isFieldFocused)

                        if let hint = rangeHint {
                            Text(hint)
                                .font(.caption)
                                .foregroundStyle(Color.secondaryLabelColor)
                        }

                        if let unit = config.unit {
                            Text("Unit: \(unit)")
                                .font(.caption)
                                .foregroundStyle(Color.tertiaryLabelColor)
                        }
                    }
                }
                .padding(.horizontal, HSpacing.screenHorizontal)
            }

            VStack {
                Divider()
                Button {
                    if let value = parsedNumber {
                        onSubmit(value)
                    }
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
                    isFieldFocused = false
                }
            }
        }
    }
}

#Preview {
    NavigationStack {
        NumberResponseView(
            request: MockData.numberRequest,
            config: try! ResponseConfigParser.parse(
                NumberConfig.self,
                from: MockData.numberConfigJSON
            )
        )
    }
}
