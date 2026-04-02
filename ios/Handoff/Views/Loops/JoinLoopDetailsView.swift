import SwiftUI

struct JoinLoopDetailsView: View {
    let inviteCode: String
    var onJoinComplete: (() -> Void)?
    @Environment(LoopsManager.self) private var loopsManager
    @Environment(\.dismiss) private var dismiss
    @State private var isJoining = false
    @State private var errorMessage: String?
    @State private var joinedLoop: APILoop?

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(spacing: 24) {
                    // Loop icon
                    Circle()
                        .fill(Color.brandGold.opacity(0.15))
                        .frame(width: 80, height: 80)
                        .overlay {
                            Image(systemName: "arrow.triangle.2.circlepath")
                                .font(.system(size: 32))
                                .foregroundStyle(Color.brandGold)
                        }
                        .padding(.top, 24)

                    VStack(spacing: 8) {
                        if let loop = joinedLoop {
                            Text(loop.name)
                                .font(.title2.bold())

                            if let desc = loop.description {
                                Text(desc)
                                    .font(.subheadline)
                                    .foregroundStyle(Color.secondaryLabelColor)
                                    .multilineTextAlignment(.center)
                            }
                        } else {
                            Text("Join Loop")
                                .font(.title2.bold())

                            Text("You're about to join a review loop")
                                .font(.subheadline)
                                .foregroundStyle(Color.secondaryLabelColor)
                        }
                    }

                    // Details
                    VStack(spacing: 0) {
                        if let loop = joinedLoop, let count = loop.memberCount {
                            detailRow(
                                icon: "person.2",
                                label: "Members",
                                value: "\(count)"
                            )
                            Divider().padding(.horizontal)
                        }
                        detailRow(
                            icon: "number",
                            label: "Invite Code",
                            value: inviteCode
                        )
                    }
                    .background(Color.secondaryBgColor)
                    .clipShape(RoundedRectangle(cornerRadius: 12))
                    .padding(.horizontal, HSpacing.screenHorizontal)

                    // What to expect
                    VStack(alignment: .leading, spacing: 12) {
                        Text("What to expect")
                            .font(.headline)

                        featureItem(
                            icon: "bell",
                            text: "Push notifications for new requests"
                        )
                        featureItem(
                            icon: "clock",
                            text: "Time-sensitive approvals and reviews"
                        )
                        featureItem(
                            icon: "hand.raised",
                            text: "Human oversight for AI workflows"
                        )
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, HSpacing.screenHorizontal)
                }
            }

            VStack(spacing: 12) {
                if let errorMessage {
                    Text(errorMessage)
                        .font(.caption)
                        .foregroundStyle(.red)
                        .padding(.horizontal, HSpacing.screenHorizontal)
                }

                Divider()
                Button {
                    joinLoop()
                } label: {
                    if isJoining {
                        ProgressView()
                            .tint(Color.bgColor)
                    } else {
                        Text("Join Loop")
                    }
                }
                .buttonStyle(PrimaryButtonStyle(isEnabled: !isJoining))
                .disabled(isJoining)
                .padding(.horizontal, HSpacing.screenHorizontal)
                .padding(.bottom, 12)
            }
            .background(Color.bgColor)
        }
        .constrainedWidth(500)
        .frame(maxWidth: .infinity)
        .navigationTitle("Loop Details")
        .inlineNavigationTitle()
    }

    private func joinLoop() {
        isJoining = true
        errorMessage = nil
        Task {
            do {
                let loop = try await loopsManager.joinLoop(inviteCode: inviteCode)
                joinedLoop = loop
                onJoinComplete?()
            } catch let error as APIError {
                if case .conflict = error {
                    // Already a member — treat as success
                    errorMessage = nil
                    onJoinComplete?()
                } else {
                    errorMessage = error.localizedDescription
                }
            } catch {
                errorMessage = error.localizedDescription
            }
            isJoining = false
        }
    }

    private func detailRow(icon: String, label: String, value: String) -> some View {
        HStack {
            Label(label, systemImage: icon)
                .font(.body)
                .foregroundStyle(Color.labelColor)

            Spacer()

            Text(value)
                .font(.body)
                .foregroundStyle(Color.secondaryLabelColor)
        }
        .padding()
    }

    private func featureItem(icon: String, text: String) -> some View {
        HStack(spacing: 12) {
            Image(systemName: icon)
                .font(.body)
                .foregroundStyle(Color.brandGold)
                .frame(width: 24)

            Text(text)
                .font(.subheadline)
                .foregroundStyle(Color.labelColor)
        }
    }
}

#Preview {
    NavigationStack {
        JoinLoopDetailsView(inviteCode: "LQGIKS")
    }
    .environment(LoopsManager())
}
