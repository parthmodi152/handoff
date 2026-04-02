import SwiftUI
#if canImport(AppKit)
import AppKit
#endif

struct SettingsView: View {
    @Environment(AuthService.self) private var authService
    @Environment(LoopsManager.self) private var loopsManager
    @Environment(PushNotificationService.self) private var pushService

    var body: some View {
        Form {
            // Profile
            Section {
                HStack(spacing: 14) {
                    Text(initials)
                        .font(.title3.bold())
                        .foregroundStyle(Color.brandGold)
                        .frame(width: 48, height: 48)
                        .background(Color.brandGold.opacity(0.15))
                        .clipShape(Circle())

                    VStack(alignment: .leading, spacing: 2) {
                        Text(authService.currentUserName)
                            .font(.body.weight(.semibold))
                        Text(authService.currentUserEmail)
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                }
                .padding(.vertical, 4)
            }

            // Loops
            Section {
                if loopsManager.loops.isEmpty {
                    Text("No loops joined yet")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(loopsManager.loops) { loop in
                        HStack {
                            Label(loop.name, systemImage: "arrow.triangle.2.circlepath")
                            Spacer()
                            if let role = loop.role {
                                Text(role.capitalized)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }
            } header: {
                Text("Loops")
            } footer: {
                #if os(iOS)
                Text("Sensitive requests from all loops appear on your Lock Screen and Dynamic Island automatically.")
                #endif
            }

            // App
            Section("App") {
                #if canImport(UIKit)
                Button {
                    if let url = URL(string: UIApplication.openNotificationSettingsURLString) {
                        UIApplication.shared.open(url)
                    }
                } label: {
                    HStack {
                        Label("Notifications", systemImage: "bell")
                        Spacer()
                        Text(pushService.permissionGranted ? "On" : "Off")
                            .foregroundStyle(.secondary)
                        Image(systemName: "arrow.up.forward")
                            .font(.caption)
                            .foregroundStyle(.tertiary)
                    }
                }
                .tint(.primary)
                #elseif canImport(AppKit)
                Button {
                    if let url = URL(string: "x-apple.systempreferences:com.apple.Notifications-Settings") {
                        NSWorkspace.shared.open(url)
                    }
                } label: {
                    HStack {
                        Label("Notifications", systemImage: "bell")
                        Spacer()
                        Text(pushService.permissionGranted ? "On" : "Off")
                            .foregroundStyle(.secondary)
                        Image(systemName: "arrow.up.forward")
                            .font(.caption)
                            .foregroundStyle(.tertiary)
                    }
                }
                .tint(.primary)
                #endif

                NavigationLink {
                    Text("Appearance")
                        .navigationTitle("Appearance")
                } label: {
                    Label("Appearance", systemImage: "paintbrush")
                }

                NavigationLink {
                    Text("About Handoff")
                        .navigationTitle("About")
                } label: {
                    Label("About", systemImage: "info.circle")
                }
            }

            // Sign Out
            Section {
                Button(role: .destructive) {
                    authService.signOut()
                } label: {
                    HStack {
                        Spacer()
                        Text("Sign Out")
                        Spacer()
                    }
                }
            }
        }
        .formStyle(.grouped)
        .navigationTitle("Settings")
        .task {
            if loopsManager.loops.isEmpty {
                await loopsManager.fetchLoops()
            }
        }
        #if canImport(UIKit)
        .task {
            for await _ in NotificationCenter.default.notifications(named: UIApplication.willEnterForegroundNotification) {
                await pushService.checkPermissionStatus()
            }
        }
        #elseif canImport(AppKit)
        .task {
            for await _ in NotificationCenter.default.notifications(named: NSApplication.didBecomeActiveNotification) {
                await pushService.checkPermissionStatus()
            }
        }
        #endif
    }

    private var initials: String {
        let parts = authService.currentUserName.split(separator: " ")
        let first = parts.first?.prefix(1) ?? ""
        let last = parts.count > 1 ? parts.last?.prefix(1) ?? "" : ""
        return "\(first)\(last)".uppercased()
    }
}

#Preview {
    NavigationStack {
        SettingsView()
    }
    .environment(MockData.previewAuthService())
    .environment(MockData.previewLoopsManager())
    .environment(PushNotificationService())
}
