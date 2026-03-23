require "test_helper"

class ApplicationHelperTest < ActionView::TestCase
  test "safe_link_href allows http and https urls" do
    assert_equal "https://example.com", safe_link_href("https://example.com")
    assert_equal "http://example.com", safe_link_href("http://example.com")
  end

  test "safe_link_href allows mailto and tel urls" do
    assert_equal "mailto:test@example.com", safe_link_href("mailto:test@example.com")
    assert_equal "tel:+15551234567", safe_link_href("tel:+15551234567")
  end

  test "safe_link_href allows relative paths" do
    assert_equal "/dashboard", safe_link_href("/dashboard")
  end

  test "safe_link_href falls back for unsafe schemes" do
    assert_equal "#", safe_link_href("javascript:alert(1)")
    assert_equal "#", safe_link_href("data:text/html;base64,SGVsbG8=")
  end

  test "safe_link_href falls back for invalid or blank input" do
    assert_equal "#", safe_link_href(nil)
    assert_equal "#", safe_link_href("   ")
    assert_equal "#", safe_link_href("http://[invalid")
  end
end
