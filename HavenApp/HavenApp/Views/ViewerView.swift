import SwiftUI
import Combine

struct ViewerView: View {
    @EnvironmentObject var configService: ConfigService
    @EnvironmentObject var nostrService: NostrService
    @EnvironmentObject var relayManager: RelayProcessManager
    
    @State private var searchText = ""
    @State private var viewMode: ViewMode = .notes
    @State private var blossomMedia: [MediaItem] = []
    @State private var selectedMedia: MediaItem? = nil
    
    enum ViewMode {
        case notes
        case media
    }
    
    var filteredNotes: [NostrEvent] {
        if searchText.isEmpty {
            return nostrService.events
        }
        return nostrService.events.filter { $0.content.localizedCaseInsensitiveContains(searchText) }
    }
    
    var allMediaItems: [MediaItem] {
        var items: [MediaItem] = []
        if relayManager.isRunning {
            items.append(contentsOf: blossomMedia)
        }
        items.append(contentsOf: nostrService.noteMedia)
        
        var uniqueItems: [MediaItem] = []
        var seenURLs = Set<URL>()
        for item in items {
            if !seenURLs.contains(item.url) {
                uniqueItems.append(item)
                seenURLs.insert(item.url)
            }
        }
        return uniqueItems.sorted(by: { $0.dateAdded > $1.dateAdded })
    }
    
    var body: some View {
        VStack(spacing: 0) {
            // MARK: - Header
            VStack(spacing: 12) {
                HStack {
                    HStack(spacing: 4) {
                        ModeButton(title: "Notes", icon: "doc.text", isSelected: viewMode == .notes) {
                            viewMode = .notes
                        }
                        ModeButton(title: "Media", icon: "photo.on.rectangle", isSelected: viewMode == .media) {
                            viewMode = .media
                            loadLocalMedia()
                        }
                    }
                    .padding(4)
                    .background(Color.primary.opacity(0.05))
                    .cornerRadius(8)
                    
                    Spacer()
                    
                    Button(action: {
                        refreshAll()
                    }) {
                        Label("Refresh", systemImage: "arrow.clockwise")
                            .font(.caption)
                    }
                    .buttonStyle(.plain)
                    .foregroundColor(.secondary)
                }
                
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("Search notes...", text: $searchText)
                        .textFieldStyle(.plain)
                }
                .padding(10)
                .background(Color(NSColor.controlBackgroundColor))
                .cornerRadius(8)
            }
            .padding()
            .background(Color(NSColor.windowBackgroundColor))
            
            Divider()
            
            // MARK: - List Content
            ScrollView {
                VStack(spacing: 0) {
                    if viewMode == .notes {
                        notesList
                    } else {
                        mediaGrid
                    }
                }
                .padding()
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .background(Color(NSColor.textBackgroundColor))
        .onAppear {
            refreshAll()
        }
        .overlay(fullScreenOverlay)
    }
    
    private var notesList: some View {
        Group {
            if filteredNotes.isEmpty {
                VStack(spacing: 16) {
                    Image(systemName: "doc.text.magnifyingglass")
                        .font(.system(size: 48))
                        .foregroundColor(.secondary)
                    Text("No notes found")
                        .font(.headline)
                    Text(relayManager.isRunning ? "Waiting for incoming events..." : "Start the relay to see notes.")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
                .padding(.top, 100)
            } else {
                LazyVStack(spacing: 12) {
                    ForEach(filteredNotes) { event in
                        NoteRow(event: event)
                    }
                }
            }
        }
    }
    
    private var mediaGrid: some View {
        let items = allMediaItems
        return Group {
            if items.isEmpty {
                VStack(spacing: 16) {
                    Image(systemName: "photo.on.rectangle")
                        .font(.system(size: 48))
                        .foregroundColor(.secondary)
                    Text("No media found")
                        .font(.headline)
                    Text("Hosted images or images in notes will appear here.")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
                .padding(.top, 100)
            } else {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 140), spacing: 12)], spacing: 12) {
                    ForEach(items) { item in
                        MediaGridItem(item: item) {
                            selectedMedia = item
                        }
                    }
                }
            }
        }
    }
    
    @ViewBuilder
    private var fullScreenOverlay: some View {
        if let item = selectedMedia {
            ZStack {
                Color.black.opacity(0.9)
                    .edgesIgnoringSafeArea(.all)
                    .onTapGesture { selectedMedia = nil }
                
                VStack {
                    HStack {
                        Button(action: {
                            NSPasteboard.general.clearContents()
                            NSPasteboard.general.setString(item.url.absoluteString, forType: .string)
                        }) {
                            Label("Copy Link", systemImage: "doc.on.doc")
                                .padding(.horizontal, 12)
                                .padding(.vertical, 8)
                                .background(Color.white.opacity(0.1))
                                .cornerRadius(8)
                        }
                        .buttonStyle(.plain)
                        
                        Spacer()
                        
                        Button(action: { selectedMedia = nil }) {
                            Image(systemName: "xmark.circle.fill")
                                .font(.title)
                                .foregroundColor(.white.opacity(0.6))
                        }
                        .buttonStyle(.plain)
                    }
                    .padding()
                    
                    Spacer()
                    
                    if item.url.isGIF {
                        AnimatedImage(url: item.url)
                            .padding()
                    } else {
                        AsyncImage(url: item.url) { phase in
                            if let image = phase.image {
                                image.resizable().aspectRatio(contentMode: .fit)
                            } else if phase.error != nil {
                                VStack(spacing: 12) {
                                    Image(systemName: "exclamationmark.triangle").font(.largeTitle)
                                    Text("Failed to load image").font(.caption)
                                }
                            } else {
                                ProgressView()
                            }
                        }
                        .padding()
                    }
                    
                    Spacer()
                    
                    Text(item.url.absoluteString)
                        .font(.caption.monospaced())
                        .foregroundColor(.secondary)
                        .padding(.bottom)
                }
            }
            .transition(.opacity)
        }
    }
    
    func refreshAll() {
        nostrService.resetConnections()
        let port = configService.config.relayPort
        let urls = [
            URL(string: "ws://localhost:\(port)/")!,
            URL(string: "ws://localhost:\(port)/inbox")!,
            URL(string: "ws://localhost:\(port)/chat")!,
            URL(string: "ws://localhost:\(port)/private")!
        ]
        nostrService.fetchNotes(from: urls)
        loadLocalMedia()
    }
    
    func loadLocalMedia() {
        let relayDataDir = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("haven_relay")
        let blossomDir = relayDataDir.appendingPathComponent("blossom")
        
        do {
            if !FileManager.default.fileExists(atPath: blossomDir.path) {
                try? FileManager.default.createDirectory(at: blossomDir, withIntermediateDirectories: true)
                self.blossomMedia = []
                return
            }
            
            let fileURLs = try FileManager.default.contentsOfDirectory(at: blossomDir, includingPropertiesForKeys: nil)
            let items = fileURLs.compactMap { url -> MediaItem? in
                let filename = url.lastPathComponent
                if filename.starts(with: ".") { return nil }
                guard let serveURL = URL(string: "http://localhost:\(configService.config.relayPort)/\(filename)") else { return nil }
                return MediaItem(id: UUID(), url: serveURL, type: .image, dateAdded: Date())
            }
            self.blossomMedia = items
        } catch {
            print("Error loading blossom media: \(error)")
        }
    }
}

struct ModeButton: View {
    let title: String
    let icon: String
    let isSelected: Bool
    let action: () -> Void
    
    var body: some View {
        Button(action: action) {
            HStack(spacing: 6) {
                Image(systemName: icon)
                Text(title)
            }
            .font(.subheadline.bold())
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
            .foregroundColor(isSelected ? .white : .secondary)
            .background(isSelected ? Color.havenPurple : Color.clear)
            .cornerRadius(20)
        }
        .buttonStyle(.plain)
    }
}

struct NoteRow: View {
    let event: NostrEvent
    
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Circle().fill(Color.havenPurple).frame(width: 24, height: 24)
                    .overlay(Image(systemName: "person.fill").font(.caption).foregroundColor(.white))
                Text(event.pubkey.prefix(8) + "..." + event.pubkey.suffix(4))
                    .font(.caption).foregroundColor(.secondary)
                Spacer()
                Text(timeAgo(from: event.createdAtDate))
                    .font(.caption).foregroundColor(.secondary)
            }
            Text(event.content).font(.body).lineLimit(10).multilineTextAlignment(.leading)
            HStack {
                HStack(spacing: 4) {
                    Image(systemName: "bubble.left")
                    Text("Kind \(event.kind)")
                }
                .font(.caption2).padding(.horizontal, 8).padding(.vertical, 4)
                .background(Color.white.opacity(0.1)).cornerRadius(4)
                Spacer()
            }
        }
        .padding()
        .background(Color(NSColor.controlBackgroundColor))
        .cornerRadius(12)
    }
    
    func timeAgo(from date: Date) -> String {
        let formatter = RelativeDateTimeFormatter()
        formatter.unitsStyle = .full
        return formatter.localizedString(for: date, relativeTo: Date())
    }
}

struct MediaGridItem: View {
    let item: MediaItem
    let onSelect: () -> Void
    
    var body: some View {
        Group {
            if item.url.isGIF {
                AnimatedImage(url: item.url)
                    .frame(minWidth: 0, maxWidth: .infinity)
                    .frame(height: 140)
                    .clipped()
            } else {
                AsyncImage(url: item.url) { phase in
                    switch phase {
                    case .empty:
                        ZStack {
                            Rectangle().fill(Color.gray.opacity(0.1))
                            ProgressView().controlSize(.small)
                        }
                    case .success(let image):
                        image.resizable().aspectRatio(contentMode: .fill)
                            .frame(minWidth: 0, maxWidth: .infinity).frame(height: 140)
                    case .failure:
                        ZStack {
                            Rectangle().fill(Color(NSColor.controlBackgroundColor))
                            VStack(spacing: 4) {
                                Image(systemName: "photo").font(.title2)
                                Text("Error").font(.caption2)
                            }
                            .foregroundColor(.secondary)
                        }
                    @unknown default: EmptyView()
                    }
                }
                .frame(height: 140).clipped()
            }
        }
        .cornerRadius(8).contentShape(Rectangle())
        .onTapGesture { onSelect() }
        .contextMenu {
            Button(action: {
                NSPasteboard.general.clearContents()
                NSPasteboard.general.setString(item.url.absoluteString, forType: .string)
            }) {
                Label("Copy Link", systemImage: "doc.on.doc")
            }
        }
    }
}
