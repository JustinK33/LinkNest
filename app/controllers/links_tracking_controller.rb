class LinksTrackingController < ApplicationController
  allow_unauthenticated_access only: :track_click
  skip_forgery_protection only: :track_click  # No CSRF check for async tracking (JS sends it)
  before_action :set_link, only: :track_click

  def track_click
    # Queue the click tracking job (async, non-blocking)
    TrackLinkClickJob.perform_later(
      @link.id,
      {
        referrer: request.referrer,
        user_agent: request.user_agent,
        ip_address: request.remote_ip,
        country_code: request.headers["CF-IPCountry"] || "US", # Cloudflare geolocation
        device_type: device_type,
        browser_name: browser_name
      }
    )

    # Respond with 202 Accepted (async operation)
    head :accepted
  end

  private

  def set_link
    # Only allow tracking for public links to prevent data leakage
    # Users can only track their own links or public links from other users
    if current_user
      # Authenticated users can track their own links (public or private) or any public link
      @link = Link.where(
        "(user_id = ? OR (public = ? OR public IS NULL))",
        current_user.id,
        true
      ).find(params[:id])
    else
      # Unauthenticated users can only track public links
      @link = Link.where(public: true).or(Link.where(public: nil)).find(params[:id])
    end
  rescue ActiveRecord::RecordNotFound
    head :not_found
  end

  def device_type
    user_agent = request.user_agent.to_s.downcase
    case user_agent
    when /mobile|android|iphone|ipod|windows phone/
      "mobile"
    when /tablet|ipad|kindle/
      "tablet"
    else
      "desktop"
    end
  end

  def browser_name
    user_agent = request.user_agent.to_s
    case user_agent
    when /chrome/i
      "Chrome"
    when /firefox/i
      "Firefox"
    when /safari/i
      "Safari"
    when /edge/i
      "Edge"
    when /opera/i
      "Opera"
    else
      "Other"
    end
  end
end
