import SwiftUI
import AVFoundation

struct JoinLoopQRView: View {
    var onCodeScanned: (String) -> Void = { _ in }
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        ZStack {
            // Camera placeholder (actual AVFoundation integration would go here)
            Color.black
                .ignoresSafeArea()

            VStack(spacing: 24) {
                Spacer()

                // Viewfinder frame
                RoundedRectangle(cornerRadius: 24)
                    .stroke(Color.white, lineWidth: 3)
                    .frame(width: 250, height: 250)
                    .overlay {
                        VStack(spacing: 12) {
                            Image(systemName: "qrcode.viewfinder")
                                .font(.system(size: 48))
                                .foregroundStyle(.white.opacity(0.6))

                            Text("Point camera at QR code")
                                .font(.subheadline)
                                .foregroundStyle(.white.opacity(0.8))
                        }
                    }

                Spacer()

                // Manual entry fallback
                Button {
                    dismiss()
                } label: {
                    Text("Enter code manually")
                        .font(.body.bold())
                        .foregroundStyle(.white)
                        .padding(.vertical, 14)
                        .frame(maxWidth: .infinity)
                        .background(.white.opacity(0.2))
                        .clipShape(RoundedRectangle(cornerRadius: HRadius.button))
                }
                .padding(.horizontal, HSpacing.screenHorizontal)
                .padding(.bottom, 32)
            }
        }
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button {
                    dismiss()
                } label: {
                    Image(systemName: "xmark")
                        .foregroundStyle(.white)
                }
            }
        }
        .navigationTitle("Scan QR")
        .inlineNavigationTitle()
        #if os(iOS) || os(visionOS)
        .toolbarColorScheme(.dark, for: .navigationBar)
        #endif
    }
}

#Preview {
    NavigationStack {
        JoinLoopQRView()
    }
}
