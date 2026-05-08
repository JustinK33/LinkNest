require "test_helper"
require "tempfile"

class LinksControllerTest < ActionDispatch::IntegrationTest
  test "create file link attaches pdf and stores blob url" do
    user = users(:one)
    sign_in_as(user)

    file = Tempfile.new([ "resume", ".pdf" ])
    file.binwrite("%PDF-1.4\n%test\n")
    file.rewind

    uploaded_file = Rack::Test::UploadedFile.new(file.path, "application/pdf")

    assert_difference("Link.count", 1) do
      post links_path, params: {
        entry_mode: "file",
        link: {
          title: "Resume",
          resume_pdf: uploaded_file
        }
      }
    end

    assert_redirected_to dashboard_path

    link = user.links.order(:created_at).last
    assert_predicate link.resume_pdf, :attached?
    assert_equal Rails.application.routes.url_helpers.rails_blob_path(link.resume_pdf, only_path: true), link.url
  ensure
    file.close
    file.unlink
  end
end
