#!/usr/bin/env ruby

ENV['RAILS_ENV'] ||= 'development'
require_relative 'config/environment'

puts "🚀 End-to-End Dashboard Test"
puts "=" * 60

# 1. Get existing test user
puts "\n1️⃣  Finding test user..."
user = User.find_by(username: "johndoe")
if user
  puts "   ✅ Found user: #{user.first_name} #{user.last_name} (@#{user.username})"
else
  puts "   ❌ User not found. Creating one..."
  user = User.create!(
    email_address: "john@example.com",
    password: "secure123",
    first_name: "John",
    last_name: "Doe",
    username: "johndoe"
  )
  puts "   ✅ Created user: #{user.first_name} #{user.last_name}"
end
puts ""

# 2. Get user's links
puts "2️⃣  Getting user's links..."
links = user.links.order(:position)
if links.any?
  puts "   ✅ Found #{links.count} links:"
  links.each { |l| puts "      - #{l.title}" }
else
  puts "   ⚠️  No links found. The dashboard will show empty state."
end
puts ""

# 3. Create realistic test click data (if doesn't exist)
puts "3️⃣  Creating realistic test clicks..."
click_count_before = user.link_clicks.count
puts "   Current clicks: #{click_count_before}"

# Create clicks spread across the last 7 days at different times
(1..35).each do |i|
  days_ago = (i / 5).to_i  # Spread across 7 days
  hours_ago = (i % 5) * 5  # Different hours

  link = links.sample
  next unless link

  LinkClick.create!(
    user_id: user.id,
    link_id: link.id,
    created_at: days_ago.days.ago - hours_ago.hours,
    referrer: [ "https://google.com", "https://twitter.com", "direct" ].sample,
    user_agent: "Mozilla/5.0",
    ip_address: "192.168.1.#{(i % 50) + 1}",
    country_code: [ "US", "UK", "CA", "DE", "FR" ].sample,
    device_type: [ "desktop", "mobile", "tablet" ].sample,
    browser_name: [ "Chrome", "Safari", "Firefox", "Edge" ].sample
  )
end

click_count_after = user.link_clicks.count
puts "   ✅ Created #{click_count_after - click_count_before} new clicks"
puts "   📊 Total clicks now: #{click_count_after}"
puts ""

# 4. Run aggregation jobs
puts "4️⃣  Running aggregation jobs..."
puts "   - Running AggregateHourlyStatsJob..."
AggregateHourlyStatsJob.perform_now
hourly_count = user.hourly_link_stats.count
puts "     ✅ Created #{hourly_count} hourly stat records"

puts "   - Running AggregateDailyStatsJob..."
AggregateDailyStatsJob.perform_now(Date.current)
daily_count = user.daily_user_stats.count
puts "     ✅ Created #{daily_count} daily stat records"
puts ""

# 5. Verify analytics data
puts "5️⃣  Verifying analytics data..."
total_clicks = user.link_clicks.count
unique_visitors = user.link_clicks.distinct.count(:ip_address)
clicks_7d = user.link_clicks.where(created_at: 7.days.ago.beginning_of_day..Time.current).count
unique_7d = user.link_clicks.where(created_at: 7.days.ago.beginning_of_day..Time.current).distinct.count(:ip_address)

puts "   📊 Analytics Summary:"
puts "      - Total clicks: #{total_clicks}"
puts "      - Unique visitors: #{unique_visitors}"
puts "      - Clicks (7d): #{clicks_7d}"
puts "      - Unique visitors (7d): #{unique_7d}"
puts ""

# 6. Verify dashboard data availability
puts "6️⃣  Dashboard Data Availability:"

daily_data = user.daily_user_stats.where(date: 7.days.ago.to_date..Date.current).count
puts "   ✅ Daily stats available: #{daily_data} days of data"

hourly_data = user.hourly_link_stats.where(created_at: 1.day.ago..Time.current).count
puts "   ✅ Hourly stats available: #{hourly_data} hours of data"

top_links_count = user.link_clicks.where(created_at: 7.days.ago.beginning_of_day..Time.current).group_by(&:link_id).count
puts "   ✅ Top links: #{top_links_count} unique links with clicks"

browsers_count = user.link_clicks.where(created_at: 7.days.ago.beginning_of_day..Time.current).group_by(&:browser_name).count
puts "   ✅ Browser breakdown: #{browsers_count} unique browsers"

devices_count = user.link_clicks.where(created_at: 7.days.ago.beginning_of_day..Time.current).group_by(&:device_type).count
puts "   ✅ Device breakdown: #{devices_count} unique devices"
puts ""

puts "=" * 60
puts "✅ All test data ready!"
puts "\n🌐 Dashboard URL: http://localhost:3000/dashboard"
puts "📧 You will need to log in first with:"
puts "   Email: #{user.email_address}"
puts "   Password: (the password you set)"
puts "\n👤 Or visit public profile: http://localhost:3000/#{user.slug}"
