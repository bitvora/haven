import Foundation

struct NostrEvent: Codable, Identifiable {
    let id: String
    let pubkey: String
    let created_at: Int64
    let kind: Int
    let tags: [[String]]
    let content: String
    let sig: String
    
    var createdAtDate: Date {
        Date(timeIntervalSince1970: TimeInterval(created_at))
    }
}
