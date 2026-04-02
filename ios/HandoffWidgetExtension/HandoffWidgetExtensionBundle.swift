//
//  HandoffWidgetExtensionBundle.swift
//  HandoffWidgetExtension
//
//  Created by Parth Modi on 2026-02-28.
//

import WidgetKit
import SwiftUI

@main
struct HandoffWidgetExtensionBundle: WidgetBundle {
    var body: some Widget {
        PendingApprovalsWidget()
        HandoffLiveActivity()
    }
}
