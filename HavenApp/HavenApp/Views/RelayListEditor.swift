import SwiftUI

struct RelayListEditor: View {
    @Binding var relays: [String]
    @State private var newRelay = ""
    
    var body: some View {
        VStack(spacing: 0) {
            List {
                ForEach(relays, id: \.self) { relay in
                    HStack {
                        Text(relay)
                        Spacer()
                        Button(action: {
                            relays.removeAll(where: { $0 == relay })
                        }) {
                            Image(systemName: "minus.circle")
                            .foregroundColor(.red)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .onDelete(perform: delete)
            }
            .listStyle(.inset)
            .frame(minHeight: 100)
            
            Divider()
            
            HStack {
                TextField("wss://relay.example.com", text: $newRelay)
                    .textFieldStyle(.plain)
                    .padding(8)
                    .onSubmit { addRelay() }
                
                Button(action: addRelay) {
                    Image(systemName: "plus.circle.fill")
                        .foregroundColor(.green)
                }
                .buttonStyle(.plain)
                .disabled(newRelay.isEmpty)
                .padding(.trailing, 8)
            }
            .background(Color(NSColor.controlBackgroundColor))
        }
        .background(Color(NSColor.controlBackgroundColor))
        .cornerRadius(8)
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(Color.gray.opacity(0.2), lineWidth: 1)
        )
    }
    
    private func addRelay() {
        var trimmed = newRelay.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty {
            if !trimmed.hasPrefix("wss://") && !trimmed.hasPrefix("ws://") {
                trimmed = "wss://" + trimmed
            }
            
            if !relays.contains(trimmed) {
                relays.append(trimmed)
                newRelay = ""
            }
        }
    }
    
    private func delete(at offsets: IndexSet) {
        relays.remove(atOffsets: offsets)
    }
}
