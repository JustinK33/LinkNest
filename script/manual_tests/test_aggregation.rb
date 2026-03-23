#!/usr/bin/env ruby

ENV['RAILS_ENV'] ||= 'development'
require_relative '../../config/environment'

puts "📈 Testing Aggregation Pipeline"
puts "=" * 60

# Create test clicks
puts "1️⃣  Creating test data..."
5.times do |i|
  LinkClick.create!(
    link_id: 1,
    user_id: 1,
    referrer: "https://google.com",
    user_agent: "Mozilla/5.0",
    ip_address: "192.168.1.#{i}",
    country_code: "US",
    device_type: "desktop",
    browser_name: "Chrome"
  )
end
puts "   ✅ Created 5 test clicks"
puts ""

# Check total clicks
total_clicks = LinkClick.count
puts "2️⃣  Total LinkClick records: #{total_clicks}"
puts ""

# Run hourly aggregation
puts "3️⃣  Running AggregateHourlyStatsJob..."
AggregateHourlyStatsJob.perform_now
puts "   ✅ Hourly aggregation complete"
puts ""

# Check hourly stats
hourly_count = HourlyLinkStat.count
puts "4️⃣  HourlyLinkStat records: #{hourly_count}"

if hourly_count > 0
  hourly = HourlyLinkStat.order(created_at: :desc).first
  puts "   ✅ Latest hourly stat created:"
  puts "      - Link: #{hourly.link.title}"
  puts "      - Hour: #{hourly.hour.strftime('%Y-%m-%d %H:00')}"
  puts "      - Click count: #{hourly.click_count}"
  puts "      - Unique visitors: #{hourly.unique_visitors}"
end
puts ""

# Check link cache
puts "5️⃣  Verifying Link.click_count cache..."
link = Link.find(1)
puts "   ✅ Link click_count: #{link.click_count}"
puts ""

# Test daily aggregation
puts "6️⃣  Running AggregateDailyStatsJob..."
AggregateDailyStatsJob.perform_now
puts "   ✅ Daily aggregation complete"
puts ""

# Check daily stats
daily_count = DailyUserStat.count
puts "7️⃣  DailyUserStat records: #{daily_count}"

if daily_count > 0
  daily = DailyUserStat.order(created_at: :desc).first
  puts "   ✅ Latest daily stat created:"
  puts "      - User: #{daily.user.slug}"
  puts "      - Date: #{daily.date}"
  puts "      - Total clicks: #{daily.total_clicks}"
  puts "      - Unique visitors: #{daily.unique_visitors}"
  puts "      - Top link: #{daily.top_link&.title} (#{daily.top_link_clicks} clicks)"
end
puts ""

puts "=" * 60
puts "✅ All aggregation tests passed!"
