import Foundation
import XCTest
@testable import Generated

final class GeneratedTests: XCTestCase {
    func testHelperRoundTripWorks() throws {
        let request = HelperAPI.GetUser.Request(userId: 7)
        let body = try HelperAPI.GetUser.encodeRequest(request)
        let bodyJSON = try JSONSerialization.jsonObject(with: body) as? [String: Any]
        XCTAssertEqual(bodyJSON?["userId"] as? NSNumber, 7)

        let response = try HelperAPI.GetUser.decodeResponse(Data(#"{"code":200,"username":"alice"}"#.utf8))
        XCTAssertEqual(response.code, 200)
        XCTAssertEqual(response.username, "alice")

        XCTAssertEqual(HelperAPI.GetUser.path, "/GetUser")
        XCTAssertEqual(HelperAPI.GetUser.urlPath, "/rpc/Helper/GetUser")
    }

    func testErrorDecodeWorks() throws {
        let error = decodeWebRPCError(
            statusCode: 404,
            data: Data(#"{"error":"UserNotFound","code":200,"msg":"user not found","cause":"","status":404}"#.utf8)
        )

        XCTAssertEqual(error.error, "UserNotFound")
        XCTAssertEqual(error.code, 200)
        XCTAssertEqual(error.message, "user not found")
        XCTAssertEqual(error.status, 404)
        switch error.kind {
        case .userNotFound:
            break
        default:
            XCTFail("expected schema error kind")
        }
    }

    func testMalformedSuccessBodyBecomesWebrpcBadResponse() async throws {
        struct BadTransport: WebRPCTransport {
            func post(
                baseURL: String,
                path: String,
                body: Data,
                headers: [String: String]
            ) async throws -> WebRPCHTTPResponse {
                WebRPCHTTPResponse(statusCode: 200, body: Data("not-json".utf8))
            }
        }

        let client = HelperClient(baseURL: "https://example.com", transport: BadTransport())

        do {
            _ = try await client.getUser(HelperAPI.GetUser.Request(userId: 1))
            XCTFail("expected error")
        } catch let error as WebRPCError {
            XCTAssertEqual(error.error, "WebrpcBadResponse")
            XCTAssertEqual(error.code, WebRPCErrorKind.webrpcBadResponse.code)
            switch error.kind {
            case .webrpcBadResponse:
                break
            default:
                XCTFail("expected webrpcBadResponse kind")
            }
            XCTAssertTrue(error.cause.contains("not-json"))
        } catch {
            XCTFail("expected WebRPCError, got \(type(of: error))")
        }
    }

    func testMalformedErrorBodyBecomesWebrpcBadResponse() throws {
        let error = decodeWebRPCError(
            statusCode: 500,
            data: Data("not-json".utf8)
        )

        XCTAssertEqual(error.error, "WebrpcBadResponse")
        XCTAssertEqual(error.code, WebRPCErrorKind.webrpcBadResponse.code)
        XCTAssertEqual(error.message, "bad response")
        XCTAssertEqual(error.status, 500)
        switch error.kind {
        case .webrpcBadResponse:
            break
        default:
            XCTFail("expected webrpcBadResponse kind")
        }
        XCTAssertTrue(error.cause.contains("not-json"))
    }

    func testClientSendsWebrpcHeaderAndParsesVersions() async throws {
        actor Recorder {
            private(set) var headers: [String: String] = [:]

            func set(headers: [String: String]) {
                self.headers = headers
            }

            func snapshot() -> [String: String] {
                headers
            }
        }

        struct CaptureTransport: WebRPCTransport {
            let recorder: Recorder

            func post(
                baseURL: String,
                path: String,
                body: Data,
                headers: [String: String]
            ) async throws -> WebRPCHTTPResponse {
                await recorder.set(headers: headers)
                return WebRPCHTTPResponse(
                    statusCode: 200,
                    body: Data(#"{"code":200,"username":"alice"}"#.utf8),
                    headers: ["webrpc": WEBRPC_HEADER_VALUE]
                )
            }
        }

        let recorder = Recorder()
        let client = HelperClient(
            baseURL: "https://example.com",
            transport: CaptureTransport(recorder: recorder),
            headers: { ["X-Test": "1"] }
        )

        let response = try await client.getUser(HelperAPI.GetUser.Request(userId: 9))
        XCTAssertEqual(response.username, "alice")

        let sentHeaders = await recorder.snapshot()
        XCTAssertEqual(sentHeaders[WEBRPC_HEADER], WEBRPC_HEADER_VALUE)
        XCTAssertEqual(sentHeaders["X-Test"], "1")

        let versions = versionFromHeader(["webrpc": WEBRPC_HEADER_VALUE])
        XCTAssertFalse(versions.webrpcGenVersion.isEmpty)
        XCTAssertEqual(versions.schemaName, "Helper")
        XCTAssertEqual(versions.schemaVersion, "v1.0.0")

        let responseVersions = versionFromHeader(
            WebRPCHTTPResponse(statusCode: 200, body: Data(), headers: ["Webrpc": WEBRPC_HEADER_VALUE])
        )
        XCTAssertEqual(responseVersions, versions)
    }

    func testVersionParsingRejectsMalformedHeaders() {
        XCTAssertEqual(versionFromHeader([:]), .empty)
        XCTAssertEqual(versionFromHeader(["Webrpc": "webrpc-gen@1;gen-swift@1"]), .empty)
        XCTAssertEqual(versionFromHeader(["Webrpc": "not-a-valid-header"]), .empty)
    }

    func testBigIntFieldsDecodeAndEncodeAsStrings() throws {
        let source = Data(#"{"amount":"9007199254740993","amounts":["1","18446744073709551616"],"maybe":"42"}"#.utf8)
        let decoded = try JSONDecoder().decode(Ledger.self, from: source)

        XCTAssertEqual(decoded.amount, "9007199254740993")
        XCTAssertEqual(decoded.amounts, ["1", "18446744073709551616"])
        XCTAssertEqual(decoded.maybe, "42")

        let encoded = try JSONEncoder().encode(decoded)
        let json = try XCTUnwrap(JSONSerialization.jsonObject(with: encoded) as? [String: Any])
        XCTAssertEqual(json["amount"] as? String, "9007199254740993")
        XCTAssertEqual(json["amounts"] as? [String], ["1", "18446744073709551616"])
        XCTAssertEqual(json["maybe"] as? String, "42")
    }

    func testLargeIntegerAnyRoundTripPreservesPrecision() throws {
        let source = Data(#"{"payload":9007199254740993}"#.utf8)
        let decoded = try JSONDecoder().decode(Box.self, from: source)

        switch decoded.payload {
        case .integer(let value):
            XCTAssertEqual(value, 9_007_199_254_740_993)
        default:
            XCTFail("expected integer payload")
        }

        let encoded = try JSONEncoder().encode(decoded)
        let encodedString = String(decoding: encoded, as: UTF8.self)
        XCTAssertTrue(encodedString.contains(#""payload":9007199254740993"#))
    }
}
