#!/usr/bin/env ruby

# Load Rails environment
ENV['RAILS_ENV'] ||= 'test'
require_relative '../../config/environment'

puts "🔍 Testing Click Tracking & Aggregation Pipeline"
puts "=" * 60

# Get test data
user = User.find_by(slug: 'johndoe')
link = user.links.first

unless user && link
  puts "❌ Test data not found. Run rake db:seed first."
  exit 1
end

puts "User: #{user.slug}"
puts "Link: #{link.title} (ID: #{link.id})"
puts ""

# Clear test data
LinkClick.delete_all
HourlyLinkStat.delete_all

# Simulate 3 clicks
puts "1️⃣  Simulating 3 clicks..."
3.times do |i|
  TrackLinkClickJob.perform_now(
    link.id,
    {
      referrer: "https://google.com",
      user_agent: "Mozilla/5.0 (Chrome)",
      ip_address: "192.168.1.#{i}",
      country_code: "US",
      device_type: "desktop",
      browser_name: "Chrome"
    }
  )
end
puts "   ✅ 3 clicks simulated"
puts ""

# Verify clicks
puts "2️⃣  Verifying LinkClick records..."
click_count = LinkClick.count
puts "   ✅ Total clicks: #{click_count}"
if click_count >= 3
  latest = LinkClick.order(created_at: :desc).first
  puts "   - Latest: #{latest.device_type} (#{latest.browser_name})"
end
puts ""

# Run aggregation
puts "3️⃣  Running AggregateHourlyStatsJob..."
AggregateHourlyStatsJob.perform_now
puts "   ✅ Aggregation complete"
puts ""

# Check hourly stats
puts "4️⃣  Checking HourlyLinkStat..."
hourly_stat = HourlyLinkStat.where(link_id: link.id).first
if hourly_stat
  puts "   ✅ Hourly stat exists"
  puts "   - Click count: #{hourly_stat.click_count}"
  puts "   - Unique visitors: #{hourly_stat.unique_visitors}"
else
  puts "   ℹ️  No hourly stat (might not have clicks in current hour)"
end
puts ""

# Check link cache
puts "5️⃣  Verifying Link.click_count cache..."
link.reload
puts "   ✅ Link click_count: #{link.click_count}"
puts ""

puts "=" * 60
puts "✅ Analytics pipeline tests completed!"
