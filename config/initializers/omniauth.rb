Rails.application.config.middleware.use OmniAuth::Builder do
  google_client_id = ENV["GOOGLE_CLIENT_ID"].to_s
  google_client_secret = ENV["GOOGLE_CLIENT_SECRET"].to_s

  if google_client_id.present? && google_client_secret.present?
    provider :google_oauth2,
      google_client_id,
      google_client_secret,
      {
        scope: "email profile",
        prompt: "select_account"
      }
  end
end

OmniAuth.config.allowed_request_methods = [ :post ]
OmniAuth.config.silence_get_warning = true
