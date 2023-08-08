import Foundation

print("Startig client...")
print("WebRPCVersion: \(WebRPCVersion)")
print("WebRPCSchemaVersion: \(WebRPCSchemaVersion)")
print("WebRPCSchemaHash: \(WebRPCSchemaHash)")
print("")

let client = ExampleServiceClient(hostname: "http://localhost:3000")

do {
    print("Sending ping...")
    try await client.ping()

    print("Getting user 1...")
    print(try await client.getUser(userID: 1))
} catch {
    print(error)
}