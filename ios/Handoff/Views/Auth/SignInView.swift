import SwiftUI
import AuthenticationServices

struct SignInView: View {
    @Environment(AuthService.self) private var authService
    @Environment(\.webAuthenticationSession) private var webAuthSession

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            // App branding
            VStack(spacing: 16) {
                Image(systemName: "hand.raised.fill")
                    .font(.system(size: 64))
                    .foregroundStyle(Color.brandGold)

                Text("Handoff")
                    .font(.largeTitle.bold())

                Text("Human oversight for AI workflows")
                    .font(.subheadline)
                    .foregroundStyle(Color.secondaryLabelColor)
            }

            Spacer()

            // Auth buttons
            VStack(spacing: 14) {
                SignInWithAppleButton(.signIn) { request in
                    let nonce = authService.prepareAppleSignIn()
                    request.requestedScopes = [.fullName, .email]
                    request.nonce = authService.sha256(nonce)
                } onCompletion: { result in
                    Task {
                        await authService.handleAppleSignIn(result: result)
                    }
                }
                .signInWithAppleButtonStyle(.black)
                .frame(height: HSize.authButtonHeight)
                .clipShape(RoundedRectangle(cornerRadius: HRadius.button))

                Button {
                    Task {
                        await authService.performOAuth(provider: "google", using: webAuthSession)
                    }
                } label: {
                    HStack(spacing: 8) {
                        Text("G")
                            .font(.title2.bold())
                            .foregroundStyle(.red)

                        Text("Sign in with Google")
                            .font(.body.bold())
                    }
                }
                .buttonStyle(SecondaryButtonStyle())
                .frame(height: HSize.authButtonHeight)
            }
            .padding(.horizontal, HSpacing.screenHorizontal)
            .padding(.bottom, 48)

            // Terms
            Text("By continuing, you agree to our Terms of Service and Privacy Policy.")
                .font(.caption2)
                .foregroundStyle(Color.tertiaryLabelColor)
                .multilineTextAlignment(.center)
                .padding(.horizontal, HSpacing.screenHorizontal)
                .padding(.bottom, 24)
        }
        .constrainedWidth(480)
        .frame(maxWidth: .infinity)
        .overlay {
            if authService.isLoading {
                Color.bgColor
                    .opacity(0.8)
                    .overlay {
                        ProgressView()
                            .controlSize(.large)
                    }
                    .ignoresSafeArea()
            }
        }
    }
}

#Preview {
    SignInView()
        .environment(AuthService())
}
