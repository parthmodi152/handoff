import SwiftUI
#if canImport(UIKit)
import UIKit
#elseif canImport(AppKit)
import AppKit
#endif

// MARK: - Brand Colors

extension Color {
    /// Gold accent — brand identity color. Defined in asset catalog with light/dark variants.
    /// Light: #C9A962, Dark: #D4B872
    static let brandGold = Color("AccentGold")

    // MARK: - Semantic System Colors

    #if canImport(UIKit)
    static let labelColor = Color(uiColor: .label)
    static let secondaryLabelColor = Color(uiColor: .secondaryLabel)
    static let tertiaryLabelColor = Color(uiColor: .tertiaryLabel)
    static let bgColor = Color(uiColor: .systemBackground)
    static let secondaryBgColor = Color(uiColor: .secondarySystemBackground)
    static let separatorColor = Color(uiColor: .separator)
    static let placeholderColor = Color(uiColor: .placeholderText)
    #elseif canImport(AppKit)
    static let labelColor = Color(nsColor: .labelColor)
    static let secondaryLabelColor = Color(nsColor: .secondaryLabelColor)
    static let tertiaryLabelColor = Color(nsColor: .tertiaryLabelColor)
    static let bgColor = Color(nsColor: .windowBackgroundColor)
    static let secondaryBgColor = Color(nsColor: .controlBackgroundColor)
    static let separatorColor = Color(nsColor: .separatorColor)
    static let placeholderColor = Color(nsColor: .placeholderTextColor)
    #endif
}

// MARK: - View Helpers

extension View {
    @ViewBuilder
    func inlineNavigationTitle() -> some View {
        #if os(iOS)
        self.navigationBarTitleDisplayMode(.inline)
        #else
        self
        #endif
    }
}

// MARK: - Content Width Constraint

extension View {
    /// Constrains content to a readable width on macOS/iPad, no-op on compact iPhone.
    @ViewBuilder
    func constrainedWidth(_ maxWidth: CGFloat = 600) -> some View {
        self.frame(maxWidth: maxWidth)
    }
}

// MARK: - Spacing Constants

enum HSpacing {
    static let screenHorizontal: CGFloat = 24
    static let cardVertical: CGFloat = 16
    static let cardHorizontal: CGFloat = 24
    static let cardGap: CGFloat = 6
    static let sectionGap: CGFloat = 16
    static let elementGap: CGFloat = 8
    static let buttonGap: CGFloat = 14
}

// MARK: - Dimension Constants

enum HSize {
    static let buttonHeight: CGFloat = 52
    static let authButtonHeight: CGFloat = 56
    static let filterHeight: CGFloat = 36
    static let optionHeight: CGFloat = 52
    static let settingsRowHeight: CGFloat = 48
    static let avatarSize: CGFloat = 64
    static let sendButtonSize: CGFloat = 52
    static let starSize: CGFloat = 36
}

enum HRadius {
    static let button: CGFloat = 16
    static let pill: CGFloat = 20
    static let codeBox: CGFloat = 12
    static let option: CGFloat = 14
    static let statusBadge: CGFloat = 8
}

// MARK: - Primary Button Style

struct PrimaryButtonStyle: ButtonStyle {
    var isEnabled: Bool = true

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.body.bold())
            .foregroundStyle(Color.bgColor)
            .frame(maxWidth: .infinity)
            .frame(height: HSize.buttonHeight)
            .background(Color.labelColor)
            .clipShape(RoundedRectangle(cornerRadius: HRadius.button))
            .opacity(isEnabled ? (configuration.isPressed ? 0.8 : 1.0) : 0.3)
    }
}

// MARK: - Secondary Button Style

struct SecondaryButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.body.bold())
            .foregroundStyle(Color.labelColor)
            .frame(maxWidth: .infinity)
            .frame(height: HSize.buttonHeight)
            .background(Color.secondaryBgColor)
            .clipShape(RoundedRectangle(cornerRadius: HRadius.button))
            .overlay(
                RoundedRectangle(cornerRadius: HRadius.button)
                    .stroke(Color.separatorColor, lineWidth: 1.5)
            )
            .opacity(configuration.isPressed ? 0.8 : 1.0)
    }
}
