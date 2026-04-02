import Foundation

// MARK: - Text Response Config

struct TextResponseConfig: Codable {
    let placeholder: String?
    let minLength: Int?
    let maxLength: Int?
    let required: Bool?
}

// MARK: - Single Select Config

struct SingleSelectConfig: Codable {
    let options: [SelectOption]
    let required: Bool?
}

// MARK: - Multi Select Config

struct MultiSelectConfig: Codable {
    let options: [SelectOption]
    let minSelections: Int?
    let maxSelections: Int?
    let required: Bool?
}

struct SelectOption: Codable, Identifiable, Hashable {
    let value: String
    let label: String
    let color: String?

    var id: String { value }
}

// MARK: - Rating Config

struct RatingConfig: Codable {
    let scaleMin: Int?
    let scaleMax: Int
    let scaleStep: Double?
    let labels: [String: String]?
    let required: Bool?

    var minValue: Int { scaleMin ?? 1 }
    var maxValue: Int { scaleMax }
    var step: Double { scaleStep ?? 1.0 }
    var minLabel: String? { labels?[String(minValue)] }
    var maxLabel: String? { labels?[String(maxValue)] }
}

// MARK: - Number Config

struct NumberConfig: Codable {
    let minValue: Double?
    let maxValue: Double?
    let decimalPlaces: Int?
    let allowNegative: Bool?
    let required: Bool?
    let unit: String?
}

// MARK: - Boolean Config

struct BooleanConfig: Codable {
    let trueLabel: String?
    let falseLabel: String?
    let trueColor: String?
    let falseColor: String?
    let required: Bool?

    var resolvedTrueLabel: String { trueLabel ?? "Yes" }
    var resolvedFalseLabel: String { falseLabel ?? "No" }
}

// MARK: - Config Parsing Helper

enum ResponseConfigParser {
    private static let decoder: JSONDecoder = {
        let d = JSONDecoder()
        d.keyDecodingStrategy = .convertFromSnakeCase
        return d
    }()

    static func parse<T: Decodable>(_ type: T.Type, from json: String) throws -> T {
        return try decoder.decode(T.self, from: Data(json.utf8))
    }

    static func parse<T: Decodable>(_ type: T.Type, from anyCodable: AnyCodable) throws -> T {
        let data = try JSONSerialization.data(withJSONObject: anyCodable.value)
        return try decoder.decode(T.self, from: data)
    }
}
