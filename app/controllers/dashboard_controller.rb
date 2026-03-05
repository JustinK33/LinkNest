class DashboardController < ApplicationController
  require "ostruct"

  def show
    @user = Current.session.user

    # Get user's real links from database
    @links = @user.links.order(:position)

    # Calculate analytics from real data
    @analytics = calculate_analytics(@user)
  end

  private

  def calculate_analytics(user)
    now = Time.current

    # Total stats (all time)
    total_clicks = user.link_clicks.count
    total_unique_visitors = user.link_clicks.distinct.count(:ip_address)

    # Last 7 days stats
    seven_days_ago = 7.days.ago.beginning_of_day
    clicks_7d = user.link_clicks.where(created_at: seven_days_ago..now).count
    unique_7d = user.link_clicks.where(created_at: seven_days_ago..now).distinct.count(:ip_address)

    # Last 24 hours stats
    one_day_ago = 1.day.ago
    clicks_24h = user.link_clicks.where(created_at: one_day_ago..now).count
    unique_24h = user.link_clicks.where(created_at: one_day_ago..now).distinct.count(:ip_address)

    # Top links in last 7 days (using Ruby to avoid complex SQL)
    top_links_data = user.link_clicks
      .where(created_at: seven_days_ago..now)
      .group_by(&:link_id)
      .sort_by { |_link_id, clicks| -clicks.length }
      .first(5)
      .map { |link_id, clicks| [ Link.find(link_id), clicks.length ] }
      .map { |link, count| OpenStruct.new(id: link.id, title: link.title, click_count: count) }

    # Daily trend data (last 7 days for chart)
    daily_data = user.daily_user_stats
      .where(date: 7.days.ago.to_date..Date.current)
      .order(date: :asc)
      .map { |stat| { date: stat.date.strftime("%b %d"), clicks: stat.total_clicks, visitors: stat.unique_visitors } }

    # Hourly trend data (last 24 hours for chart)
    hourly_data = user.hourly_link_stats
      .where(created_at: one_day_ago..now)
      .order(hour: :asc)
      .group_by { |stat| stat.hour.beginning_of_hour }
      .map { |hour, stats| { time: hour.strftime("%l %p"), clicks: stats.sum(&:click_count) } }

    # Browser/Device breakdown (last 7 days)
    browsers = user.link_clicks
      .where(created_at: seven_days_ago..now)
      .group_by(&:browser_name)
      .sort_by { |_name, clicks| -clicks.length }
      .first(5)
      .map { |name, clicks| OpenStruct.new(browser_name: name || "Unknown", click_count: clicks.length) }

    device_types = user.link_clicks
      .where(created_at: seven_days_ago..now)
      .group_by(&:device_type)
      .sort_by { |_type, clicks| -clicks.length }
      .first(5)
      .map { |type, clicks| OpenStruct.new(device_type: type || "Unknown", click_count: clicks.length) }

    {
      total_clicks: total_clicks,
      total_unique_visitors: total_unique_visitors,
      clicks_24h: clicks_24h,
      unique_24h: unique_24h,
      clicks_7d: clicks_7d,
      unique_7d: unique_7d,
      top_links: top_links_data,
      daily_data: daily_data,
      hourly_data: hourly_data,
      browsers: browsers,
      device_types: device_types
    }
  end
end
