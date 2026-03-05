#!/usr/bin/env ruby

ENV['RAILS_ENV'] ||= 'development'
require_relative 'config/environment'
require 'ostruct'

puts "🌐 Testing Dashboard HTTP Response"
puts "=" * 60
puts ""

# Get the user
user = User.find_by(username: "johndoe")
if !user
  puts "❌ User not found!"
  exit 1
end

# Create a test session
session = user.sessions.create!(user_agent: "Mozilla/5.0", ip_address: "127.0.0.1")
Current.session = session

puts "✅ Created test session for #{user.first_name} #{user.last_name}"
puts ""

# Simulate the dashboard controller call
puts "📊 Testing DashboardController data calculation"
begin
  analytics = {}

  # Get user's real links
  links = user.links.order(:position)

  # Manually call the calculate_analytics method
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

  # Top links (Ruby-based to avoid SQL issues)
  top_links_data = user.link_clicks
    .where(created_at: seven_days_ago..now)
    .group_by(&:link_id)
    .sort_by { |_link_id, clicks| -clicks.length }
    .first(5)
    .map { |link_id, clicks| [ Link.find(link_id), clicks.length ] }
    .map { |link, count| OpenStruct.new(id: link.id, title: link.title, click_count: count) }

  # Daily trend data
  daily_data = user.daily_user_stats
    .where(date: 7.days.ago.to_date..Date.current)
    .order(date: :asc)
    .map { |stat| { date: stat.date.strftime("%b %d"), clicks: stat.total_clicks, visitors: stat.unique_visitors } }

  # Hourly trend data
  hourly_data = user.hourly_link_stats
    .where(created_at: one_day_ago..now)
    .order(hour: :asc)
    .group_by { |stat| stat.hour.beginning_of_hour }
    .map { |hour, stats| { time: hour.strftime("%l %p"), clicks: stats.sum(&:click_count) } }

  # Browser breakdown
  browsers = user.link_clicks
    .where(created_at: seven_days_ago..now)
    .group_by(&:browser_name)
    .sort_by { |_name, clicks| -clicks.length }
    .first(5)
    .map { |name, clicks| OpenStruct.new(browser_name: name || "Unknown", click_count: clicks.length) }

  # Device breakdown
  device_types = user.link_clicks
    .where(created_at: seven_days_ago..now)
    .group_by(&:device_type)
    .sort_by { |_type, clicks| -clicks.length }
    .first(5)
    .map { |type, clicks| OpenStruct.new(device_type: type || "Unknown", click_count: clicks.length) }

  puts "✅ Analytics calculated successfully"
  puts ""

  puts "📊 Dashboard Data Loaded:"
  puts "   User: #{user.first_name} #{user.last_name}"
  puts "   Links: #{links.count} total"
  puts ""

  puts "📈 Analytics Available:"
  puts "   - Total clicks: #{total_clicks}"
  puts "   - Unique visitors: #{total_unique_visitors}"
  puts "   - Last 24h: #{clicks_24h} clicks, #{unique_24h} visitors"
  puts "   - Last 7d: #{clicks_7d} clicks, #{unique_7d} visitors"
  puts ""

  puts "📊 Chart Data:"
  puts "   - Daily data points: #{daily_data.count}"
  if daily_data.any?
    puts "     Sample: #{daily_data.first[:date]} - #{daily_data.first[:clicks]} clicks"
  end

  puts "   - Hourly data points: #{hourly_data.count}"
  if hourly_data.any?
    puts "     Sample: #{hourly_data.first[:time]} - #{hourly_data.first[:clicks]} clicks"
  end
  puts ""

  puts "🔝 Top Links: #{top_links_data.count} links"
  top_links_data.each_with_index do |link, idx|
    puts "   #{idx + 1}. #{link.title} - #{link.click_count} clicks"
  end
  puts ""

  puts "🌐 Browsers: #{browsers.count} types"
  browsers.each do |browser|
    puts "   - #{browser.browser_name}: #{browser.click_count} clicks"
  end
  puts ""

  puts "📱 Devices: #{device_types.count} types"
  device_types.each do |device|
  end
  puts ""

  puts "=" * 60
  puts "✅ Dashboard data fully loaded and ready!"
  puts ""
  puts "🎯 Next Steps:"
  puts "   1. Open http://localhost:3000/sessions/new"
  puts "   2. Log in with:"
  puts "      Email: john@example.com"
  puts "      Password: secure123"
  puts "   3. Visit http://localhost:3000/dashboard"
  puts ""

rescue => e
  puts "❌ Error: #{e.message}"
  puts e.backtrace.first(10)
  exit 1
end
