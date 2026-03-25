class DashboardController < ApplicationController
  def show
    @user = Current.session.user
    @links = @user.links.with_attached_resume_pdf.order(:position)

    # Generate QR code for user's public profile with error handling
    begin
      profile_url = user_profile_url(@user)
      qr_code = RQRCode::QRCode.new(profile_url)
      @qr_png = qr_code.as_png(
        resize_gte_to: false,
        resize_exactly_to: 300,
        fill: "white",
        color: "black",
        border_modules: 2
      )
    rescue RQRCode::QRCodeRunTimeError => e
      Rails.logger.error "QR Code generation failed: #{e.message}"
      @qr_error = "Unable to generate QR code at this time"
      @qr_png = nil
    rescue StandardError => e
      Rails.logger.error "Unexpected error generating QR code: #{e.message}"
      @qr_error = "Unable to generate QR code at this time"
      @qr_png = nil
    end
  end
end
