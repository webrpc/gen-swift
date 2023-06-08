import XCTest
@testable import gen_swift

final class gen_swiftTests: XCTestCase {

    private let client = TestApiClient(hostname: "http://localhost:9988")

    func testEmpty() async throws {
        await XCTAssertNoThrow(try await client.getEmpty(), "getEmpty() should get empty type successfully")
    }

    func testError() async throws {
        await XCTAssertThrowsError(try await client.getError(), "getError() should throw error")
    }

    func testOne() async throws {
        let response = try await client.getOne()
        await XCTAssertNoThrow(try await client.sendOne(one: response),
                               "getOne() should receive simple type and send it back via sendOne() successfully")

    }

    func testMulti() async throws {
        let (one, two, three) = try await client.getMulti()
        await XCTAssertNoThrow(try await client.sendMulti(one: one, two: two, three: three),
                               "getMulti() should receive simple type and send it back via sendMulti() successfully")
    }

    func testComplex() async throws {
        let response = try await client.getComplex()
        await XCTAssertNoThrow(try await client.sendComplex(complex: response),
                               "getComplex() should receive complex type and send it back via sendComplex() successfully")
    }

    func testCustomErrors() async  throws {
        let errors: [WebrpcError] = [
            .init(
                error: "WebrpcEndpoint",
                code: 0,
                message: "endpoint error",
                cause: "failed to read file: unexpected EOF",
                status: 400,
                errorKind: .webrpcEndpointError
            ),
            .init(
                error: "Unauthorized",
                code: 1,
                message: "unauthorized",
                cause: "failed to verify JWT token",
                status: 401,
                errorKind: .unauthorizedError
            ),
            .init(
                error: "ExpiredToken",
                code: 2,
                message: "expired token",
                cause: nil,
                status: 401,
                errorKind: .expiredTokenError
            ),
            .init(
                error: "InvalidToken",
                code: 3,
                message: "invalid token",
                cause: nil,
                status: 401,
                errorKind: .invalidTokenError
            ),
            .init(
                error: "Deactivated",
                code: 4,
                message: "account deactivated",
                cause: nil,
                status: 403,
                errorKind: .deactivatedError
            ),
            .init(
                error: "ConfirmAccount",
                code: 5,
                message: "confirm your email",
                cause: nil,
                status: 403,
                errorKind: .confirmAccountError
            ),
            .init(
                error: "AccessDenied",
                code: 6,
                message: "access denied",
                cause: nil,
                status: 403,
                errorKind: .accessDeniedError
            ),
            .init(
                error: "MissingArgument",
                code: 7,
                message: "missing argument",
                cause: nil,
                status: 400,
                errorKind: .missingArgumentError
            ),
            .init(
                error: "UnexpectedValue",
                code: 8,
                message: "unexpected value",
                cause: nil,
                status: 400,
                errorKind: .unexpectedValueError
            ),
            .init(
                error: "RateLimited",
                code: 100,
                message: "too many requests",
                cause: "1000 req/min exceeded",
                status: 429,
                errorKind: .rateLimitedError
            ),
            .init(
                error: "DatabaseDown",
                code: 101,
                message: "service outage",
                cause: nil,
                status: 503,
                errorKind: .databaseDownError
            ),
            .init(
                error: "ElasticDown",
                code: 102,
                message: "search is degraded",
                cause: nil,
                status: 503,
                errorKind: .elasticDownError
            ),
            .init(
                error: "NotImplemented",
                code: 103,
                message: "not implemented",
                cause: nil,
                status: 501,
                errorKind: .notImplementedError
            ),
            .init(
                error: "UserNotFound",
                code: 200,
                message: "user not found",
                cause: nil,
                status: 400,
                errorKind: .userNotFoundError
            ),
            .init(
                error: "UserBusy",
                code: 201,
                message: "user busy",
                cause: nil,
                status: 400,
                errorKind: .userBusyError
            ),
            .init(
                error: "InvalidUsername",
                code: 202,
                message: "invalid username",
                cause: nil,
                status: 400,
                errorKind: .invalidUsernameError
            ),
            .init(
                error: "FileTooBig",
                code: 300,
                message: "file is too big (max 1GB)",
                cause: nil,
                status: 400,
                errorKind: .fileTooBigError
            ),
            .init(
                error: "FileInfected",
                code: 301,
                message: "file is infected",
                cause: nil,
                status: 400,
                errorKind: .fileInfectedError
            ),
            .init(
                error: "FileType",
                code: 302,
                message: "unsupported file type",
                cause: ".wav is not supported",
                status: 400,
                errorKind: .fileTypeError
            )
        ]
        for error in errors {
            do {
                try await client.getSchemaError(code: error.code)
                XCTFail("Expected to throw \(error)")
            } catch let err as WebrpcError {
                XCTAssertEqual(error.code, err.code)
                XCTAssertEqual(error.error, err.error)
                XCTAssertEqual(error.message, err.message)
                XCTAssertEqual(error.status, err.status)
                XCTAssertEqual(error.cause, err.cause)
                XCTAssertEqual(error.kind, err.kind)
            } catch let err {
                XCTFail("Expected to throw \(error) but got \(err) instead")
            }
        }
    }
}

extension XCTest {
    func XCTAssertThrowsError<T: Sendable>(
        _ expression: @autoclosure () async throws -> T,
        _ message: @autoclosure () -> String = "",
        file: StaticString = #filePath,
        line: UInt = #line,
        _ errorHandler: (_ error: Error) -> Void = { _ in }
    ) async {
        do {
            _ = try await expression()
            XCTFail(message(), file: file, line: line)
        } catch {
            errorHandler(error)
        }
    }

    func XCTAssertNoThrow<T: Sendable>(
        _ expression: @autoclosure () async throws -> T,
        _ message: @autoclosure () -> String = "",
        file: StaticString = #filePath,
        line: UInt = #line
    ) async {
        do {
            _ = try await expression()
        } catch {
            XCTFail(message(), file: file, line: line)
        }
    }
}
