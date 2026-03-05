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
    @link = Link.find(params[:id])
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
