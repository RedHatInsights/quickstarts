#!/usr/bin/env python3
"""
Disaster Recovery Test Script for Quickstarts Favorites
Toggles a quickstart favorite every 15 seconds to console.stage.redhat.com
Uses offline token from access.redhat.com for authentication
"""

import argparse
import base64
import json
import os
import sys
import time
import requests
from datetime import datetime


class QuickstartsFavoriteTester:
    """
    Disaster recovery tester for Quickstarts API favorite functionality.

    Continuously toggles the favorite state of a quickstart to test API resilience
    and generate sustained load for disaster recovery testing.

    Args:
        base_url: Base URL for the Quickstarts API (e.g., https://console.stage.redhat.com)
        offline_token: Offline refresh token from access.redhat.com for SSO authentication
        sso_url: SSO token refresh endpoint URL
        quickstart_name: Optional quickstart name to toggle (fetches from API if not provided)
        proxy: Optional HTTP/HTTPS proxy URL
    """

    def __init__(self, base_url, offline_token, sso_url, quickstart_name=None, proxy=None):
        self.base_url = base_url.rstrip('/')
        self.offline_token = offline_token
        self.sso_url = sso_url
        self.favorite_state = False
        self.proxies = {'http': proxy, 'https': proxy} if proxy else None
        self.token_refresh_time = None

        # Exchange offline token for access token
        print("[INFO] Exchanging offline token for access token...")
        self.access_token = self._refresh_access_token()

        # Extract account_id from access token
        self.account_id = self._extract_account_from_jwt()

        # Fetch a quickstart from the API if not provided
        self.quickstart_name = quickstart_name or self._fetch_available_quickstart()

    def _refresh_access_token(self):
        """
        Exchange offline token for access token via SSO.

        Posts to the SSO refresh endpoint with the offline token to obtain a new
        short-lived access token (expires in ~5 minutes).

        Returns:
            str: SSO access token

        Raises:
            SystemExit: If token exchange fails or returns invalid response
        """
        try:
            payload = {
                'grant_type': 'refresh_token',
                'client_id': 'rhsm-api',
                'refresh_token': self.offline_token
            }

            print(f"[INFO] Requesting access token from {self.sso_url}")
            response = requests.post(
                self.sso_url,
                data=payload,
                proxies=self.proxies,
                timeout=30
            )

            if response.status_code == 200:
                data = response.json()
                access_token = data.get('access_token')
                if not access_token:
                    print("[ERROR] No access_token in SSO response")
                    sys.exit(1)
                self.token_refresh_time = time.time()
                print("[INFO] Successfully obtained access token")
                return access_token
            else:
                print(f"[ERROR] Failed to refresh token: {response.status_code}")
                print(f"[ERROR] Response: {response.text}")
                sys.exit(1)

        except Exception as e:
            print(f"[ERROR] Failed to refresh access token: {e}")
            import traceback
            traceback.print_exc()
            sys.exit(1)

    def _extract_account_from_jwt(self):
        """
        Extract account_id from JWT access token.

        Decodes the JWT payload using base64url encoding (RFC 7515) and extracts
        the account_id field, falling back to the sub field if not present.

        Returns:
            str: Account ID extracted from JWT payload

        Raises:
            SystemExit: If JWT is malformed or decoding fails
        """
        try:
            clean_token = ''.join(self.access_token.split())
            parts = clean_token.split('.')

            if len(parts) != 3:
                print("[ERROR] Invalid JWT token format")
                sys.exit(1)

            # Decode the payload (second part) using base64url encoding (RFC 7515)
            payload = parts[1]
            padding = 4 - len(payload) % 4
            if padding != 4:
                payload += '=' * padding

            decoded = base64.urlsafe_b64decode(payload)
            jwt_data = json.loads(decoded)

            # Extract account_id from JWT payload
            account_id = jwt_data.get('account_id') or jwt_data.get('sub')
            print(f"[INFO] Extracted account_id from JWT: {account_id}")
            return account_id

        except Exception as e:
            print(f"[ERROR] Failed to extract account from JWT: {e}")
            import traceback
            traceback.print_exc()
            sys.exit(1)

    def _get_auth_headers(self):
        """
        Get authentication headers for API requests.

        Returns:
            dict: Headers with Bearer token authentication and JSON content type
        """
        return {
            'Authorization': f'Bearer {self.access_token}',
            'Content-Type': 'application/json'
        }

    def _get_request_id(self, response):
        """
        Extract request ID from response headers.

        Checks multiple common header names used for request tracking in Red Hat services.

        Args:
            response: HTTP response object

        Returns:
            str or None: Request ID if found, None otherwise
        """
        for header in ['x-rh-insights-request-id', 'x-rh-request-id', 'x-request-id']:
            if header in response.headers:
                return response.headers[header]
        return None

    def _fetch_available_quickstart(self):
        """
        Fetch an available quickstart from the API.

        Queries the Quickstarts API for the user's account and selects the first
        available quickstart to use for testing.

        Returns:
            str: Name of the first available quickstart

        Raises:
            SystemExit: If API request fails or no quickstarts are available
        """
        url = f"{self.base_url}/api/quickstarts/v1/quickstarts?account={self.account_id}"

        print("[INIT] Fetching available quickstarts from API...")
        print(f"[REQUEST] GET {url}")

        try:
            response = requests.get(
                url,
                headers=self._get_auth_headers(),
                proxies=self.proxies,
                timeout=30
            )

            request_id = self._get_request_id(response)
            if request_id:
                print(f"[REQUEST-ID] {request_id}")

            if response.status_code == 200:
                data = response.json()
                quickstarts = data.get('data', [])

                if not quickstarts:
                    print("[ERROR] No quickstarts found in API response")
                    sys.exit(1)

                quickstart_name = quickstarts[0].get('name')
                if not quickstart_name:
                    print("[ERROR] Quickstart data missing 'name' field")
                    sys.exit(1)

                print(f"[INIT] Selected quickstart: {quickstart_name}")
                return quickstart_name
            else:
                print(f"[ERROR] Failed to fetch quickstarts: {response.status_code}")
                print(f"[ERROR] Response: {response.text}")
                sys.exit(1)

        except Exception as e:
            print(f"[ERROR] Request failed: {e}")
            import traceback
            traceback.print_exc()
            sys.exit(1)

    def toggle_favorite(self):
        """
        Toggle the favorite state of the quickstart.

        Alternates between favoriting and unfavoriting the quickstart by POSTing
        to the favorites API endpoint. Logs the request details and response status.
        """
        timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')

        self.favorite_state = not self.favorite_state
        action = "FAVORITE" if self.favorite_state else "UNFAVORITE"

        url = f"{self.base_url}/api/quickstarts/v1/favorites?account={self.account_id}"
        payload = {
            "quickstartName": self.quickstart_name,
            "favorite": self.favorite_state
        }

        print(f"\n[{timestamp}] {action}: {self.quickstart_name}")
        print(f"[REQUEST] POST {url}")
        print(f"[PAYLOAD] {json.dumps(payload, indent=2)}")

        try:
            response = requests.post(
                url,
                headers=self._get_auth_headers(),
                json=payload,
                proxies=self.proxies,
                timeout=30
            )

            request_id = self._get_request_id(response)
            if request_id:
                print(f"[REQUEST-ID] {request_id}")

            print(f"[RESPONSE] Status: {response.status_code}")

            if response.status_code == 200:
                try:
                    data = response.json()
                    print(f"[SUCCESS] {json.dumps(data, indent=2)}")
                except json.JSONDecodeError:
                    print(f"[SUCCESS] Response body: {response.text}")
            else:
                print(f"[ERROR] {response.status_code}: {response.text}")

        except Exception as e:
            print(f"[ERROR] Request failed: {e}")

    def run_continuous(self, interval=15):
        """
        Run the toggle operation continuously in an infinite loop.

        Executes the favorite toggle operation at regular intervals while
        automatically refreshing the access token every 4 minutes to prevent
        authentication failures (tokens expire after ~5 minutes).

        Args:
            interval: Time in seconds between toggle operations (default: 15)

        Raises:
            SystemExit: On KeyboardInterrupt (Ctrl+C)
        """
        print(f"=== Quickstarts Disaster Recovery Test ===")
        print(f"Target: {self.base_url}")
        print(f"Quickstart: {self.quickstart_name}")
        print(f"Account: {self.account_id}")
        print(f"Interval: {interval}s")
        print("===")

        try:
            iteration = 0
            # Refresh token every 4 minutes (tokens expire in ~5 minutes)
            TOKEN_REFRESH_INTERVAL = 240  # 4 minutes in seconds

            while True:
                iteration += 1
                print(f"\n{'='*60}")
                print(f"Iteration #{iteration}")
                print(f"{'='*60}")

                # Check if we need to refresh the token
                if time.time() - self.token_refresh_time >= TOKEN_REFRESH_INTERVAL:
                    print("\n[INFO] Refreshing access token (tokens expire after ~5 minutes)...")
                    try:
                        self.access_token = self._refresh_access_token()
                    except Exception as e:
                        print(f"[ERROR] Failed to refresh token: {e}")
                        print("[WARN] Continuing with existing token, may fail on next request")

                self.toggle_favorite()

                print(f"\n[SLEEP] Waiting {interval} seconds until next toggle...")
                time.sleep(interval)

        except KeyboardInterrupt:
            print("\n\n[STOPPED] Test interrupted by user")
            sys.exit(0)


def main():
    """
    Main entry point for the disaster recovery test script.

    Parses command-line arguments, initializes the QuickstartsFavoriteTester,
    and starts the continuous toggle operation.

    Raises:
        SystemExit: If offline token is not provided or initialization fails
    """
    parser = argparse.ArgumentParser(
        description='Disaster Recovery Test - Toggle Quickstart Favorites',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Basic usage with offline token file
  python3 disaster_recovery_test.py --offline-token-file /tmp/token.txt

  # With custom interval (30 seconds)
  python3 disaster_recovery_test.py --offline-token-file /tmp/token.txt --interval 30

  # With proxy
  python3 disaster_recovery_test.py --offline-token-file /tmp/token.txt \\
    --proxy http://squid.corp.redhat.com:3128

  # Specify a specific quickstart
  python3 disaster_recovery_test.py --offline-token-file /tmp/token.txt \\
    --quickstart insights-getting-started

  # Using environment variables
  export OFFLINE_TOKEN=$(cat /tmp/token.txt)
  export SSO_REFRESH_TOKEN_URL="https://sso.stage.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token"
  python3 disaster_recovery_test.py

How to get an offline token:
  1. Visit https://access.stage.redhat.com/management/api
  2. Generate an offline token
  3. Save it to a file (e.g., /tmp/token.txt)
  4. Run the script with --offline-token-file /tmp/token.txt
        """
    )

    parser.add_argument(
        '--base-url',
        default=os.environ.get('CONSOLEDOT_BASE_URL', 'https://console.stage.redhat.com'),
        help='Base URL for the quickstarts API (default: $CONSOLEDOT_BASE_URL or https://console.stage.redhat.com)'
    )
    parser.add_argument(
        '--offline-token',
        default=os.environ.get('OFFLINE_TOKEN', ''),
        help='Offline token from access.redhat.com/management/api (default: $OFFLINE_TOKEN)'
    )
    parser.add_argument(
        '--offline-token-file',
        help='Path to file containing offline token (alternative to --offline-token)'
    )
    parser.add_argument(
        '--sso-url',
        default=os.environ.get('SSO_REFRESH_TOKEN_URL', 'https://sso.stage.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token'),
        help='SSO token refresh URL (default: $SSO_REFRESH_TOKEN_URL or stage SSO URL)'
    )
    parser.add_argument(
        '--quickstart',
        default=None,
        help='Quickstart name to toggle (default: fetch first available from API)'
    )
    parser.add_argument(
        '--interval',
        type=int,
        default=15,
        help='Interval in seconds between toggles (default: 15)'
    )
    parser.add_argument(
        '--proxy',
        help='HTTP/HTTPS proxy URL (e.g., http://squid.corp.redhat.com:3128)'
    )

    args = parser.parse_args()

    # Read offline token from file if provided
    if args.offline_token_file:
        try:
            with open(args.offline_token_file, 'r') as f:
                args.offline_token = f.read().strip()
            print(f"[INFO] Read offline token from {args.offline_token_file}")
        except Exception as e:
            print(f"[ERROR] Failed to read offline token file: {e}")
            sys.exit(1)

    # Validate that we have an offline token
    if not args.offline_token:
        print("[ERROR] No offline token provided.")
        print("[ERROR] Use --offline-token-file or set $OFFLINE_TOKEN environment variable")
        print("\nHow to get an offline token:")
        print("1. Visit https://access.stage.redhat.com/management/api")
        print("2. Generate an offline token")
        print("3. Save it to a file and run:")
        print("   python3 disaster_recovery_test.py --offline-token-file /path/to/token.txt")
        sys.exit(1)

    tester = QuickstartsFavoriteTester(
        base_url=args.base_url,
        offline_token=args.offline_token,
        sso_url=args.sso_url,
        quickstart_name=args.quickstart,
        proxy=args.proxy
    )

    tester.run_continuous(interval=args.interval)


if __name__ == '__main__':
    main()
