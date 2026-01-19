import Foundation
import Combine

class WebSocketClient: NSObject, ObservableObject, URLSessionWebSocketDelegate {
    private var webSocketTask: URLSessionWebSocketTask?
    @Published var isConnected = false
    let messageSubject = PassthroughSubject<String, Never>()
    
    private static let session: URLSession = {
        let config = URLSessionConfiguration.default
        return URLSession(configuration: config)
    }()
    
    func connect(url: URL) {
        disconnect()
        let task = Self.session.webSocketTask(with: url)
        task.delegate = self
        webSocketTask = task
        task.resume()
        receiveMessage()
    }
    
    func disconnect() {
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
        isConnected = false
    }
    
    func send(text: String) {
        let message = URLSessionWebSocketTask.Message.string(text)
        webSocketTask?.send(message) { error in
            if let error = error {
                print("WebSocket send error: \(error)")
            }
        }
    }
    
    private func receiveMessage() {
        guard let task = webSocketTask else { return }
        task.receive { [weak self] result in
            switch result {
            case .failure(let error):
                print("WebSocket receive failure: \(error.localizedDescription)")
                DispatchQueue.main.async { self?.isConnected = false }
            case .success(let message):
                switch message {
                case .string(let text):
                    self?.messageSubject.send(text)
                case .data(_):
                    break
                @unknown default:
                    break
                }
                self?.receiveMessage()
            }
        }
    }
    
    // MARK: - URLSessionWebSocketDelegate
    func urlSession(_ session: URLSession, webSocketTask: URLSessionWebSocketTask, didOpenWithProtocol protocol: String?) {
        DispatchQueue.main.async { self.isConnected = true }
    }
    
    func urlSession(_ session: URLSession, webSocketTask: URLSessionWebSocketTask, didCloseWith closeCode: URLSessionWebSocketTask.CloseCode, reason: Data?) {
        DispatchQueue.main.async { self.isConnected = false }
    }

    func urlSession(_ session: URLSession, task: URLSessionTask, didCompleteWithError error: Error?) {
        if let _ = error {
            DispatchQueue.main.async { self.isConnected = false }
        }
    }
}

class NostrService: ObservableObject {
    // These are no longer @Published to prevent background-thread notification crashes.
    // We notify manually on the main thread via the throttled subject.
    private(set) var events: [NostrEvent] = []
    private(set) var noteMedia: [MediaItem] = []
    
    private var seenEventIds = Set<String>()
    private var clients: [String: WebSocketClient] = [:]
    private var cancellables = Set<AnyCancellable>()
    private let processingQueue = DispatchQueue(label: "com.haven.nostr-processing", qos: .userInitiated)
    
    // Batching updates to the UI
    private let eventUpdateSubject = PassthroughSubject<Void, Never>()
    
    init() {
        setupThrottling()
    }
    
    func resetConnections() {
        for client in clients.values {
            client.disconnect()
        }
        clients.removeAll()
        cancellables.removeAll()
        setupThrottling()
    }
    
    private func setupThrottling() {
        // Debounce UI updates to prevent main thread saturation and fix NSStatusItem threading crash
        eventUpdateSubject
            .throttle(for: .milliseconds(300), scheduler: DispatchQueue.main, latest: true)
            .sink { [weak self] in
                self?.objectWillChange.send()
            }
            .store(in: &cancellables)
    }
    
    func fetchNotes(from relayURLs: [URL]) {
        for url in relayURLs {
            if clients[url.absoluteString] != nil { continue }
            
            let client = WebSocketClient()
            clients[url.absoluteString] = client
            
            client.messageSubject
                .receive(on: processingQueue)
                .sink { [weak self] message in
                    self?.processMessage(message)
                }
                .store(in: &cancellables)
            
            client.$isConnected
                .receive(on: DispatchQueue.main)
                .sink { isConnected in
                    if isConnected {
                        self.sendRequest(to: client, url: url)
                    }
                }
                .store(in: &cancellables)
            
            client.connect(url: url)
        }
    }
    
    private func sendRequest(to client: WebSocketClient, url: URL) {
        let subscriptionId = "viewer-\(url.lastPathComponent.isEmpty ? "root" : url.lastPathComponent)-\(UUID().uuidString.prefix(4))"
        // Request KIND 1 (notes) and KIND 1063 (file metadata)
        let req = ["REQ", subscriptionId, ["kinds": [1, 1063], "limit": 500]] as [Any]
        if let reqData = try? JSONSerialization.data(withJSONObject: req),
           let reqString = String(data: reqData, encoding: .utf8) {
            client.send(text: reqString)
        }
    }
    
    private func processMessage(_ message: String) {
        guard let data = message.data(using: .utf8),
              let json = try? JSONSerialization.jsonObject(with: data) as? [Any],
              json.count >= 3,
              let type = json[0] as? String, type == "EVENT" else { return }
        
        if let eventDict = json[2] as? [String: Any],
           let eventData = try? JSONSerialization.data(withJSONObject: eventDict),
           let event = try? JSONDecoder().decode(NostrEvent.self, from: eventData) {
            
            if seenEventIds.contains(event.id) { return }
            seenEventIds.insert(event.id)
            
            var items: [MediaItem] = []
            
            if event.kind == 1063 {
                // Parse KIND 1063 url tag
                if let urlTag = event.tags.first(where: { $0.count >= 2 && $0[0] == "url" }),
                   let url = URL(string: urlTag[1]) {
                    items.append(MediaItem(id: UUID(), url: url, type: .image, dateAdded: event.createdAtDate))
                }
            } else {
                let urls = extractMediaURLs(from: event.content)
                items = urls.map { MediaItem(id: UUID(), url: $0, type: .image, dateAdded: event.createdAtDate) }
            }
            
            // Perform sorting and limiting on background queue
            processingQueue.async { [weak self] in
                guard let self = self else { return }
                
                if event.kind == 1 {
                    self.events.append(event)
                }
                
                if !items.isEmpty {
                    self.noteMedia.append(contentsOf: items)
                }
                
                // Only sort occasionally or if we have a burst finished
                if self.events.count % 20 == 0 || items.count > 0 {
                    self.events.sort(by: { $0.created_at > $1.created_at })
                    if self.events.count > 1000 {
                        self.events = Array(self.events.prefix(1000))
                    }
                }
                
                // Signal UI update - this will be throttled and sent on main thread
                self.eventUpdateSubject.send()
            }
        }
    }
    
    func extractMediaURLs(from content: String) -> [URL] {
        // More robust pattern for media URLs, including those without extensions but following common patterns
        let pattern = #"(https?://\S+?\.(?:jpg|jpeg|png|gif|webp|mp4|mov|webm)(?:\?\S+)?)|(https?://\S+?/blossom/[a-f0-9]{64})"#
        guard let regex = try? NSRegularExpression(pattern: pattern, options: .caseInsensitive) else { return [] }
        let nsString = content as NSString
        let results = regex.matches(in: content, options: [], range: NSRange(location: 0, length: nsString.length))
        
        return results.compactMap { result in
            let urlString = nsString.substring(with: result.range)
            return URL(string: urlString)
        }
    }
}
