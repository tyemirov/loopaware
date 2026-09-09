import Foundation

enum MetadataError: Error, CustomStringConvertible {
    case invalid(String)

    var description: String {
        switch self {
        case .invalid(let message): return message
        }
    }
}

let servicesKey = "NSBonjourServices"
let descriptionKey = "NSLocalNetworkUsageDescription"
let developmentService = "_expo._tcp"
let developmentDescription = "Expo Dev Launcher uses the local network to discover and connect to development servers running on your computer."

do {
    guard CommandLine.arguments.count == 2 else {
        throw MetadataError.invalid("expected the built Info.plist path")
    }
    let url = URL(fileURLWithPath: CommandLine.arguments[1])
    let input = try Data(contentsOf: url)
    var format = PropertyListSerialization.PropertyListFormat.xml
    let propertyList = try PropertyListSerialization.propertyList(from: input, options: [], format: &format)
    guard var metadata = propertyList as? [String: Any] else {
        throw MetadataError.invalid("Info.plist must contain a dictionary")
    }
    var changed = false
    if let value = metadata[servicesKey] {
        guard let services = value as? [String] else {
            throw MetadataError.invalid("NSBonjourServices must contain an array of strings")
        }
        let retained = services.filter { $0.lowercased().replacingOccurrences(of: "\\.$", with: "", options: .regularExpression) != developmentService }
        if retained.isEmpty {
            metadata.removeValue(forKey: servicesKey)
            changed = true
        } else if retained != services {
            metadata[servicesKey] = retained
            changed = true
        }
    }
    if let value = metadata[descriptionKey] {
        guard let description = value as? String else {
            throw MetadataError.invalid("NSLocalNetworkUsageDescription must contain a string")
        }
        if description == developmentDescription {
            metadata.removeValue(forKey: descriptionKey)
            changed = true
        }
    }
    if changed {
        let output = try PropertyListSerialization.data(fromPropertyList: metadata, format: format, options: 0)
        try output.write(to: url, options: .atomic)
    }
} catch {
    FileHandle.standardError.write(Data("Apple release metadata cleanup failed: \(error)\n".utf8))
    exit(1)
}
