import SwiftUI

struct JoinLoopCodeView: View {
    var onJoinComplete: (() -> Void)?
    @State private var code: [String] = Array(repeating: "", count: 6)
    @State private var focusedIndex: Int = 0
    @State private var showQRScanner = false
    @State private var showLoopDetails = false
    @State private var combinedCode: String = ""
    @FocusState private var isFieldFocused: Bool
    @Environment(\.dismiss) private var dismiss

    private var fullCode: String {
        code.joined()
    }

    private var isCodeComplete: Bool {
        fullCode.count == 6
    }

    var body: some View {
        VStack(spacing: 32) {
            VStack(spacing: 8) {
                Text("Join a Loop")
                    .font(.title2.bold())

                Text("Enter the 6-character invite code")
                    .font(.subheadline)
                    .foregroundStyle(Color.secondaryLabelColor)
            }
            .padding(.top, 24)

            // Code input boxes
            codeBoxes

            #if os(iOS) || os(visionOS)
            Divider()
                .padding(.horizontal, HSpacing.screenHorizontal)

            Button {
                showQRScanner = true
            } label: {
                Label("Scan QR Code", systemImage: "qrcode.viewfinder")
            }
            .buttonStyle(SecondaryButtonStyle())
            .padding(.horizontal, HSpacing.screenHorizontal)
            #endif

            Spacer()
        }
        .padding(.horizontal, HSpacing.screenHorizontal)
        #if os(macOS)
        .frame(minWidth: 400, minHeight: 300)
        #endif
        .onAppear {
            isFieldFocused = true
        }
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button("Cancel") {
                    dismiss()
                }
            }
        }
        .navigationTitle("Join Loop")
        .inlineNavigationTitle()
        #if os(iOS) || os(visionOS)
        .sheet(isPresented: $showQRScanner) {
            NavigationStack {
                JoinLoopQRView { scannedCode in
                    combinedCode = scannedCode.uppercased()
                    showQRScanner = false
                    showLoopDetails = true
                }
            }
        }
        #endif
        .navigationDestination(isPresented: $showLoopDetails) {
            JoinLoopDetailsView(inviteCode: fullCode, onJoinComplete: onJoinComplete)
        }
    }

    // MARK: - Code Boxes

    private var codeBoxesContent: some View {
        HStack(spacing: 10) {
            ForEach(0..<6, id: \.self) { index in
                Text(code[index].isEmpty ? " " : code[index])
                    .font(.title.monospacedDigit().bold())
                    .frame(width: 48, height: 56)
                    .background(Color.secondaryBgColor)
                    .clipShape(RoundedRectangle(cornerRadius: HRadius.codeBox))
                    .overlay(
                        RoundedRectangle(cornerRadius: HRadius.codeBox)
                            .stroke(
                                index == focusedIndex
                                    ? Color.labelColor
                                    : Color.separatorColor,
                                lineWidth: index == focusedIndex ? 2 : 1
                            )
                    )
            }
        }
    }

    @ViewBuilder
    private var codeBoxes: some View {
        #if os(macOS)
        // On macOS, use onKeyPress to avoid the NSTextField focus ring entirely
        codeBoxesContent
            .focusable()
            .focused($isFieldFocused)
            .focusEffectDisabled()
            .onKeyPress(characters: .alphanumerics, phases: .down) { press in
                let char = press.characters.uppercased()
                guard !char.isEmpty else { return .ignored }
                let currentCount = code.filter { !$0.isEmpty }.count
                if currentCount < 6 {
                    code[currentCount] = String(char.prefix(1))
                    focusedIndex = min(currentCount + 1, 5)
                    if currentCount + 1 == 6 {
                        showLoopDetails = true
                    }
                }
                return .handled
            }
            .onKeyPress(.delete) {
                let currentCount = code.filter { !$0.isEmpty }.count
                if currentCount > 0 {
                    code[currentCount - 1] = ""
                    focusedIndex = max(currentCount - 2, 0)
                }
                return .handled
            }
            .contentShape(Rectangle())
            .onTapGesture {
                isFieldFocused = true
            }
        #else
        // On iOS, use a hidden TextField to trigger the software keyboard
        codeBoxesContent
            .background {
                TextField("", text: $combinedCode)
                    .keyboardType(.asciiCapable)
                    .textInputAutocapitalization(.characters)
                    .autocorrectionDisabled()
                    .focused($isFieldFocused)
                    .frame(width: 0, height: 0)
                    .opacity(0)
                    .onChange(of: combinedCode) { _, newValue in
                        let cleaned = String(newValue.prefix(6)).uppercased()
                        for (i, char) in cleaned.enumerated() {
                            code[i] = String(char)
                        }
                        for i in cleaned.count..<6 {
                            code[i] = ""
                        }
                        focusedIndex = min(cleaned.count, 5)
                        combinedCode = cleaned

                        if cleaned.count == 6 {
                            showLoopDetails = true
                        }
                    }
            }
            .contentShape(Rectangle())
            .onTapGesture {
                isFieldFocused = true
            }
        #endif
    }
}

#Preview {
    NavigationStack {
        JoinLoopCodeView()
    }
    .environment(LoopsManager())
}
