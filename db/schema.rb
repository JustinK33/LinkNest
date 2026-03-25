# This file is auto-generated from the current state of the database. Instead
# of editing this file, please use the migrations feature of Active Record to
# incrementally modify your database, and then regenerate this schema definition.
#
# This file is the source Rails uses to define your schema when running `bin/rails
# db:schema:load`. When creating a new database, `bin/rails db:schema:load` tends to
# be faster and is potentially less error prone than running all of your
# migrations from scratch. Old migrations may fail to apply correctly if those
# migrations use external dependencies or application code.
#
# It's strongly recommended that you check this file into your version control system.

ActiveRecord::Schema[8.1].define(version: 2026_03_25_051616) do
  create_table "action_text_rich_texts", charset: "utf8mb4", collation: "utf8mb4_bin", force: :cascade do |t|
    t.text "body", size: :long
    t.datetime "created_at", null: false
    t.string "name", null: false
    t.bigint "record_id", null: false
    t.string "record_type", null: false
    t.datetime "updated_at", null: false
    t.index ["record_type", "record_id", "name"], name: "index_action_text_rich_texts_uniqueness", unique: true
  end

  create_table "active_storage_attachments", charset: "utf8mb4", collation: "utf8mb4_bin", force: :cascade do |t|
    t.bigint "blob_id", null: false
    t.datetime "created_at", null: false
    t.string "name", null: false
    t.bigint "record_id", null: false
    t.string "record_type", null: false
    t.index ["blob_id"], name: "index_active_storage_attachments_on_blob_id"
    t.index ["record_type", "record_id", "name", "blob_id"], name: "index_active_storage_attachments_uniqueness", unique: true
  end

  create_table "active_storage_blobs", charset: "utf8mb4", collation: "utf8mb4_bin", force: :cascade do |t|
    t.bigint "byte_size", null: false
    t.string "checksum"
    t.string "content_type"
    t.datetime "created_at", null: false
    t.string "filename", null: false
    t.string "key", null: false
    t.text "metadata"
    t.string "service_name", null: false
    t.index ["key"], name: "index_active_storage_blobs_on_key", unique: true
  end

  create_table "active_storage_variant_records", charset: "utf8mb4", collation: "utf8mb4_bin", force: :cascade do |t|
    t.bigint "blob_id", null: false
    t.string "variation_digest", null: false
    t.index ["blob_id", "variation_digest"], name: "index_active_storage_variant_records_uniqueness", unique: true
  end

  create_table "daily_user_stats", charset: "utf8mb4", collation: "utf8mb4_bin", force: :cascade do |t|
    t.datetime "created_at", null: false
    t.date "date", null: false
    t.integer "top_link_clicks", default: 0
    t.integer "top_link_id"
    t.integer "total_clicks", default: 0
    t.integer "unique_visitors", default: 0
    t.datetime "updated_at", null: false
    t.bigint "user_id", null: false
    t.index ["date"], name: "index_daily_user_stats_on_date"
    t.index ["user_id", "date"], name: "index_daily_user_stats_on_user_id_and_date"
    t.index ["user_id"], name: "index_daily_user_stats_on_user_id"
  end

  create_table "hourly_link_stats", charset: "utf8mb4", collation: "utf8mb4_bin", force: :cascade do |t|
    t.integer "click_count", default: 0
    t.datetime "created_at", null: false
    t.datetime "hour", null: false
    t.bigint "link_id", null: false
    t.integer "unique_visitors", default: 0
    t.datetime "updated_at", null: false
    t.bigint "user_id", null: false
    t.index ["hour"], name: "index_hourly_link_stats_on_hour"
    t.index ["link_id", "hour"], name: "index_hourly_link_stats_on_link_id_and_hour", unique: true
    t.index ["link_id"], name: "index_hourly_link_stats_on_link_id"
    t.index ["user_id", "hour"], name: "index_hourly_link_stats_on_user_id_and_hour"
    t.index ["user_id"], name: "index_hourly_link_stats_on_user_id"
  end

  create_table "link_clicks", charset: "utf8mb4", collation: "utf8mb4_bin", force: :cascade do |t|
    t.string "browser_name"
    t.string "country_code"
    t.datetime "created_at", null: false
    t.string "device_type"
    t.string "ip_address"
    t.bigint "link_id", null: false
    t.string "referrer"
    t.datetime "updated_at", null: false
    t.string "user_agent"
    t.bigint "user_id", null: false
    t.index ["created_at"], name: "index_link_clicks_on_created_at"
    t.index ["link_id", "created_at"], name: "index_link_clicks_on_link_id_and_created_at"
    t.index ["link_id"], name: "index_link_clicks_on_link_id"
    t.index ["user_id", "created_at"], name: "index_link_clicks_on_user_id_and_created_at"
    t.index ["user_id"], name: "index_link_clicks_on_user_id"
  end

  create_table "links", charset: "utf8mb4", collation: "utf8mb4_bin", force: :cascade do |t|
    t.integer "click_count", default: 0
    t.datetime "created_at", null: false
    t.string "icon_color", default: "#3b82f6"
    t.integer "inventory_count", default: 0
    t.integer "position", default: 0
    t.boolean "public", default: true, null: false
    t.string "title"
    t.datetime "updated_at", null: false
    t.string "url"
    t.bigint "user_id"
    t.index ["user_id", "created_at"], name: "index_links_on_user_id_and_created_at"
    t.index ["user_id", "position"], name: "index_links_on_user_id_and_position"
    t.index ["user_id", "public", "position"], name: "index_links_on_user_id_and_public_and_position"
    t.index ["user_id"], name: "index_links_on_user_id"
  end

  create_table "sessions", charset: "utf8mb4", collation: "utf8mb4_bin", force: :cascade do |t|
    t.datetime "created_at", null: false
    t.string "ip_address"
    t.datetime "updated_at", null: false
    t.string "user_agent"
    t.bigint "user_id", null: false
    t.index ["user_id"], name: "index_sessions_on_user_id"
  end

  create_table "subscribers", charset: "utf8mb4", collation: "utf8mb4_bin", force: :cascade do |t|
    t.datetime "created_at", null: false
    t.string "email"
    t.bigint "link_id", null: false
    t.datetime "updated_at", null: false
    t.index ["link_id"], name: "index_subscribers_on_link_id"
  end

  create_table "users", charset: "utf8mb4", collation: "utf8mb4_bin", force: :cascade do |t|
    t.string "avatar_url"
    t.text "bio"
    t.datetime "created_at", null: false
    t.string "email"
    t.string "email_address", null: false
    t.string "first_name", null: false
    t.string "last_name", null: false
    t.string "password_digest", null: false
    t.string "phone_number"
    t.string "profile_color", default: "#3b82f6"
    t.string "slug", null: false
    t.datetime "updated_at", null: false
    t.string "username", null: false
    t.index ["email_address"], name: "index_users_on_email_address", unique: true
    t.index ["slug"], name: "index_users_on_slug", unique: true
    t.index ["username"], name: "index_users_on_username", unique: true
  end

  add_foreign_key "active_storage_attachments", "active_storage_blobs", column: "blob_id"
  add_foreign_key "active_storage_variant_records", "active_storage_blobs", column: "blob_id"
  add_foreign_key "daily_user_stats", "users"
  add_foreign_key "hourly_link_stats", "links"
  add_foreign_key "hourly_link_stats", "users"
  add_foreign_key "link_clicks", "links"
  add_foreign_key "link_clicks", "users"
  add_foreign_key "links", "users"
  add_foreign_key "sessions", "users"
  add_foreign_key "subscribers", "links"
end
