class ApplicationController < ActionController::Base
  include Authentication
  helper_method :google_last_used?
  # Only allow modern browsers supporting webp images, web push, badges, import maps, CSS nesting, and CSS :has.
  allow_browser versions: :modern

  # Changes to the importmap will invalidate the etag for HTML responses
  stale_when_importmap_changes

  around_action :switch_locale

  def switch_locale(&action)
    locale = params[:locale] || I18n.default_locale
    I18n.with_locale(locale, &action)
  end

  def google_last_used?
    cookies.signed[:last_auth_provider].to_s == "google_oauth2"
  end
end
