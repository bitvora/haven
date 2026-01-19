import SwiftUI
import AppKit

struct AnimatedImage: NSViewRepresentable {
    let url: URL
    
    func makeNSView(context: Context) -> NSImageView {
        let imageView = NSImageView()
        // Use scaleProportionallyUpOrDown for "aspect fit" behavior
        // To get "aspect fill", we'd need more complex logic, let's start with proper fit.
        imageView.imageScaling = .scaleProportionallyUpOrDown
        imageView.animates = true
        
        // Ensure it respects the frame given by SwiftUI
        imageView.setContentHuggingPriority(.defaultLow, for: .horizontal)
        imageView.setContentHuggingPriority(.defaultLow, for: .vertical)
        imageView.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        imageView.setContentCompressionResistancePriority(.defaultLow, for: .vertical)
        
        // Clipping at the NSView level
        imageView.wantsLayer = true
        imageView.layer?.masksToBounds = true
        
        loadAsync(url: url, into: imageView)
        return imageView
    }
    
    func updateNSView(_ nsView: NSImageView, context: Context) {
        // Handle URL changes if necessary, but for grid items the URL is usually static
    }
    
    private func loadAsync(url: URL, into imageView: NSImageView) {
        URLSession.shared.dataTask(with: url) { data, response, error in
            guard let data = data, error == nil else {
                return
            }
            
            DispatchQueue.main.async {
                if let image = NSImage(data: data) {
                    imageView.image = image
                }
            }
        }.resume()
    }
}

// Helper to determine if a URL represents a GIF
extension URL {
    var isGIF: Bool {
        return self.pathExtension.lowercased() == "gif"
    }
}
