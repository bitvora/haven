import Foundation

struct MediaItem: Identifiable, Codable {
    let id: UUID
    let url: URL
    let type: MediaType
    let dateAdded: Date
    
    enum MediaType: String, Codable {
        case image
        case video
    }
}
