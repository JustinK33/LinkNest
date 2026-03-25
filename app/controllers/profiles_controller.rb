class ProfilesController < ApplicationController
  allow_unauthenticated_access
  before_action :set_user, only: [ :show, :qr_code ]

  def show
    # Use safe scope that handles missing public column and eager load attachments
    @links = @user.links.public_links.with_attached_resume_pdf.order(:position)
  end

  def qr_code
    profile_url = user_profile_url(@user)

    begin
      qr_code = RQRCode::QRCode.new(profile_url)

      png = qr_code.as_png(
        resize_gte_to: false,
        resize_exactly_to: 400,
        fill: "white",
        color: "black",
        border_modules: 4
      )

      send_data png.to_s,
                type: "image/png",
                disposition: "inline",
                filename: "#{@user.slug}_qr_code.png"
    rescue RQRCode::QRCodeRunTimeError => e
      Rails.logger.error "QR Code generation failed for user #{@user.slug}: #{e.message}"
      redirect_to user_profile_path(@user), alert: "Unable to generate QR code at this time"
    rescue StandardError => e
      Rails.logger.error "Unexpected error generating QR code for user #{@user.slug}: #{e.message}"
      redirect_to user_profile_path(@user), alert: "Unable to generate QR code at this time"
    end
  end

  private
    def set_user
      @user = User.find_by_slug!(params[:slug])
    rescue ActiveRecord::RecordNotFound
      redirect_to root_path, alert: "Profile not found"
    end
end
