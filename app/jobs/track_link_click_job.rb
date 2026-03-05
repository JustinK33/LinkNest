class TrackLinkClickJob < ApplicationJob
  queue_as :default

  def perform(link_id, visitor_info = {})
    link = Link.find_by(id: link_id)
    return unless link

    # Create raw click record
    LinkClick.create(
      user_id: link.user_id,
      link_id: link_id,
      referrer: visitor_info[:referrer],
      user_agent: visitor_info[:user_agent],
      ip_address: visitor_info[:ip_address],
      country_code: visitor_info[:country_code],
      device_type: visitor_info[:device_type],
      browser_name: visitor_info[:browser_name]
    )
  end
end
